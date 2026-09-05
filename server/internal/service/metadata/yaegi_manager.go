package metadata

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"

	modelmd "itcodex/server/internal/model/metadata"
	yaegictx "itcodex/server/pkg/yaegi/context"
	yaegiutils "itcodex/server/pkg/yaegi/utils"
)

type YaegiInstance struct {
	interpreter *interp.Interpreter
	script      *modelmd.YaegiScript
	hooks       map[HookPoint]reflect.Value
	hasHandle   bool
	handleFn    reflect.Value
}

type DefaultYaegiManager struct {
	mu              sync.RWMutex
	db              *Database
	scripts         map[int64]*YaegiInstance
	hookMap         map[string]map[HookPoint][]*YaegiInstance
	customAPIRoutes map[string]*YaegiInstance
	yaegiPath       string
}

func NewYaegiManager(db *Database) *DefaultYaegiManager {
	return &DefaultYaegiManager{
		db:              db,
		scripts:         make(map[int64]*YaegiInstance),
		hookMap:         make(map[string]map[HookPoint][]*YaegiInstance),
		customAPIRoutes: make(map[string]*YaegiInstance),
		yaegiPath:       "./_pkg",
	}
}

func (m *DefaultYaegiManager) LoadScript(script *modelmd.YaegiScript) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if script.Id > 0 {
		m.removeScriptLocked(script.Id)
	}

	inst, err := m.compileScript(script)
	if err != nil {
		return err
	}

	if script.Id > 0 {
		m.scripts[script.Id] = inst
	}

	hp := HookPoint(script.HookPoint)
	if hp == HookPointCustomAPI && script.HTTPMethod != "" && script.APIPath != "" {
		key := strings.ToUpper(script.HTTPMethod) + ":" + script.APIPath
		m.customAPIRoutes[key] = inst
	} else if script.CollectionName != "" {
		if _, ok := m.hookMap[script.CollectionName]; !ok {
			m.hookMap[script.CollectionName] = make(map[HookPoint][]*YaegiInstance)
		}
		m.hookMap[script.CollectionName][hp] = append(m.hookMap[script.CollectionName][hp], inst)
		sort.Slice(m.hookMap[script.CollectionName][hp], func(i, j int) bool {
			return m.hookMap[script.CollectionName][hp][i].script.Priority < m.hookMap[script.CollectionName][hp][j].script.Priority
		})
	}

	return nil
}

func (m *DefaultYaegiManager) DisableScript(id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.removeScriptLocked(id)
	return nil
}

func (m *DefaultYaegiManager) removeScriptLocked(id int64) {
	inst, ok := m.scripts[id]
	if !ok {
		return
	}

	delete(m.scripts, id)

	hp := HookPoint(inst.script.HookPoint)
	if hp == HookPointCustomAPI {
		key := strings.ToUpper(inst.script.HTTPMethod) + ":" + inst.script.APIPath
		delete(m.customAPIRoutes, key)
	} else if inst.script.CollectionName != "" {
		if hooks, ok := m.hookMap[inst.script.CollectionName]; ok {
			if list, ok := hooks[hp]; ok {
				for i, item := range list {
					if item == inst {
						m.hookMap[inst.script.CollectionName][hp] = append(list[:i], list[i+1:]...)
						break
					}
				}
			}
		}
	}
}

func (m *DefaultYaegiManager) ValidateScript(content string) error {
	i := interp.New(interp.Options{GoPath: m.yaegiPath, Unrestricted: false})
	if err := i.Use(stdlib.Symbols); err != nil {
		return err
	}
	if err := i.Use(yaegictx.Symbols); err != nil {
		return err
	}
	if err := i.Use(m.buildExports()); err != nil {
		return err
	}
	if err := i.Use(yaegiutils.Symbols); err != nil {
		return err
	}
	_, err := i.Eval(content)
	return err
}

func (m *DefaultYaegiManager) FindCustomAPI(method, path string) *modelmd.YaegiScript {
	m.mu.RLock()
	defer m.mu.RUnlock()
	key := strings.ToUpper(method) + ":" + path
	if inst, ok := m.customAPIRoutes[key]; ok {
		return inst.script
	}
	return nil
}

func (m *DefaultYaegiManager) ExecuteCustomAPI(script *modelmd.YaegiScript, ctx *yaegictx.YaegiHTTPContext) (err error) {
	m.mu.RLock()
	inst, ok := m.scripts[script.Id]
	m.mu.RUnlock()

	if !ok {
		inst, err = m.compileScript(script)
		if err != nil {
			return err
		}
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("script panic: %v", r)
		}
	}()

	fn := inst.handleFn
	if !fn.IsValid() {
		return fmt.Errorf("script does not export Handle function")
	}

	args := []reflect.Value{reflect.ValueOf(ctx)}
	results := fn.Call(args)
	if len(results) > 0 {
		if errVal, ok := results[len(results)-1].Interface().(error); ok && errVal != nil {
			return errVal
		}
	}
	return nil
}

func (m *DefaultYaegiManager) ExecuteBeforeCreate(ctx context.Context, coll *Collection, data map[string]interface{}) (result map[string]interface{}, err error) {
	result = data
	hooks := m.getHooks(coll.Name(), HookPointBeforeCreate)
	for _, inst := range hooks {
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("script panic: %v", r)
				}
			}()
			if err != nil {
				return
			}
			fn, ok := inst.hooks[HookPointBeforeCreate]
			if !ok || !fn.IsValid() {
				return
			}
			args := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(result)}
			returns := fn.Call(args)
			if len(returns) >= 2 {
				if newData, ok := returns[0].Interface().(map[string]interface{}); ok && newData != nil {
					result = newData
				}
				if hookErr, ok := returns[1].Interface().(error); ok && hookErr != nil {
					err = hookErr
				}
			}
		}()
	}
	return result, err
}

func (m *DefaultYaegiManager) ExecuteAfterCreate(ctx context.Context, coll *Collection, record *Record) (err error) {
	hooks := m.getHooks(coll.Name(), HookPointAfterCreate)
	for _, inst := range hooks {
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("script panic: %v", r)
				}
			}()
			if err != nil {
				return
			}
			fn, ok := inst.hooks[HookPointAfterCreate]
			if !ok || !fn.IsValid() {
				return
			}
			args := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(record.Data())}
			returns := fn.Call(args)
			if len(returns) > 0 {
				if hookErr, ok := returns[0].Interface().(error); ok && hookErr != nil {
					err = hookErr
				}
			}
		}()
	}
	return err
}

func (m *DefaultYaegiManager) ExecuteBeforeUpdate(ctx context.Context, coll *Collection, data map[string]interface{}, filter Filter) (result map[string]interface{}, err error) {
	result = data
	hooks := m.getHooks(coll.Name(), HookPointBeforeUpdate)
	for _, inst := range hooks {
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("script panic: %v", r)
				}
			}()
			if err != nil {
				return
			}
			fn, ok := inst.hooks[HookPointBeforeUpdate]
			if !ok || !fn.IsValid() {
				return
			}
			args := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(result)}
			returns := fn.Call(args)
			if len(returns) >= 2 {
				if newData, ok := returns[0].Interface().(map[string]interface{}); ok && newData != nil {
					result = newData
				}
				if hookErr, ok := returns[1].Interface().(error); ok && hookErr != nil {
					err = hookErr
				}
			}
		}()
	}
	return result, err
}

func (m *DefaultYaegiManager) ExecuteAfterUpdate(ctx context.Context, coll *Collection, records []*Record) (err error) {
	hooks := m.getHooks(coll.Name(), HookPointAfterUpdate)
	for _, inst := range hooks {
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("script panic: %v", r)
				}
			}()
			if err != nil {
				return
			}
			fn, ok := inst.hooks[HookPointAfterUpdate]
			if !ok || !fn.IsValid() {
				return
			}
			dataList := make([]map[string]interface{}, 0, len(records))
			for _, r := range records {
				dataList = append(dataList, r.Data())
			}
			args := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(dataList)}
			returns := fn.Call(args)
			if len(returns) > 0 {
				if hookErr, ok := returns[0].Interface().(error); ok && hookErr != nil {
					err = hookErr
				}
			}
		}()
	}
	return err
}

func (m *DefaultYaegiManager) ExecuteBeforeDelete(ctx context.Context, coll *Collection, filter Filter) (err error) {
	hooks := m.getHooks(coll.Name(), HookPointBeforeDelete)
	for _, inst := range hooks {
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("script panic: %v", r)
				}
			}()
			if err != nil {
				return
			}
			fn, ok := inst.hooks[HookPointBeforeDelete]
			if !ok || !fn.IsValid() {
				return
			}
			args := []reflect.Value{reflect.ValueOf(ctx)}
			returns := fn.Call(args)
			if len(returns) > 0 {
				if hookErr, ok := returns[0].Interface().(error); ok && hookErr != nil {
					err = hookErr
				}
			}
		}()
	}
	return err
}

func (m *DefaultYaegiManager) ExecuteAfterDelete(ctx context.Context, coll *Collection, affected int) (err error) {
	hooks := m.getHooks(coll.Name(), HookPointAfterDelete)
	for _, inst := range hooks {
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("script panic: %v", r)
				}
			}()
			if err != nil {
				return
			}
			fn, ok := inst.hooks[HookPointAfterDelete]
			if !ok || !fn.IsValid() {
				return
			}
			args := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(affected)}
			returns := fn.Call(args)
			if len(returns) > 0 {
				if hookErr, ok := returns[0].Interface().(error); ok && hookErr != nil {
					err = hookErr
				}
			}
		}()
	}
	return err
}

func (m *DefaultYaegiManager) getHooks(collName string, hp HookPoint) []*YaegiInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if collHooks, ok := m.hookMap[collName]; ok {
		if list, ok := collHooks[hp]; ok {
			result := make([]*YaegiInstance, len(list))
			copy(result, list)
			return result
		}
	}
	return nil
}

func (m *DefaultYaegiManager) compileScript(script *modelmd.YaegiScript) (*YaegiInstance, error) {
	i := interp.New(interp.Options{GoPath: m.yaegiPath, Unrestricted: false})

	if err := i.Use(stdlib.Symbols); err != nil {
		return nil, err
	}

	if err := i.Use(yaegictx.Symbols); err != nil {
		return nil, err
	}
	if err := i.Use(m.buildExports()); err != nil {
		return nil, err
	}
	if err := i.Use(yaegiutils.Symbols); err != nil {
		return nil, err
	}

	if _, err := i.Eval(script.Content); err != nil {
		return nil, fmt.Errorf("script compile error: %w", err)
	}

	inst := &YaegiInstance{
		interpreter: i,
		script:      script,
		hooks:       make(map[HookPoint]reflect.Value),
	}

	hookPoints := []HookPoint{
		HookPointBeforeCreate,
		HookPointAfterCreate,
		HookPointBeforeUpdate,
		HookPointAfterUpdate,
		HookPointBeforeDelete,
		HookPointAfterDelete,
		HookPointBeforeFind,
		HookPointAfterFind,
	}

	for _, hp := range hookPoints {
		for _, name := range hookSymbolNames(hp) {
			v, err := i.Eval(name)
			if err == nil && v.IsValid() {
				inst.hooks[hp] = v
				break
			}
		}
	}

	v, err := i.Eval("Handle")
	if err == nil && v.IsValid() {
		inst.hasHandle = true
		inst.handleFn = v
	}

	return inst, nil
}

func hookSymbolNames(hp HookPoint) []string {
	s := string(hp)
	if s == "" {
		return nil
	}
	// Scripts conventionally export PascalCase (BeforeCreate); hook point is camelCase (beforeCreate).
	pascal := strings.ToUpper(s[:1]) + s[1:]
	return []string{s, pascal}
}

func (m *DefaultYaegiManager) ExecuteAfterFind(ctx context.Context, coll *Collection, records []*Record) (err error) {
	hooks := m.getHooks(coll.Name(), HookPointAfterFind)
	for _, inst := range hooks {
		func() {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("script panic: %v", r)
				}
			}()
			if err != nil {
				return
			}
			fn, ok := inst.hooks[HookPointAfterFind]
			if !ok || !fn.IsValid() {
				return
			}
			args := []reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(records)}
			returns := fn.Call(args)
			if len(returns) > 0 {
				if hookErr, ok := returns[0].Interface().(error); ok && hookErr != nil {
					err = hookErr
				}
			}
		}()
	}
	return err
}
