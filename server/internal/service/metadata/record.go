package metadata

import (
	"time"

	"github.com/gogf/gf/v2/os/gtime"
	"github.com/spf13/cast"
)

type Record struct {
	data map[string]any
}

func NewRecord(data map[string]any) *Record {
	if data == nil {
		data = make(map[string]any)
	}
	return &Record{
		data: data,
	}
}

func (r *Record) Id() int64 {
	return cast.ToInt64(r.data[DefaultPrimaryKey])
}

func (r *Record) Get(key string) any {
	if r.data == nil {
		return nil
	}
	return r.data[key]
}

func (r *Record) Set(key string, val any) {
	if r.data == nil {
		r.data = make(map[string]any)
	}
	r.data[key] = val
}

func (r *Record) Data() map[string]any {
	if r.data == nil {
		return make(map[string]any)
	}
	return r.data
}

func (r *Record) GetString(key string) string {
	return cast.ToString(r.Get(key))
}

func (r *Record) GetInt(key string) int {
	return cast.ToInt(r.Get(key))
}

func (r *Record) GetInt32(key string) int32 {
	return cast.ToInt32(r.Get(key))
}

func (r *Record) GetInt64(key string) int64 {
	return cast.ToInt64(r.Get(key))
}

func (r *Record) GetUint(key string) uint {
	return cast.ToUint(r.Get(key))
}

func (r *Record) GetUint32(key string) uint32 {
	return cast.ToUint32(r.Get(key))
}

func (r *Record) GetUint64(key string) uint64 {
	return cast.ToUint64(r.Get(key))
}

func (r *Record) GetFloat32(key string) float32 {
	return cast.ToFloat32(r.Get(key))
}

func (r *Record) GetFloat64(key string) float64 {
	return cast.ToFloat64(r.Get(key))
}

func (r *Record) GetBool(key string) bool {
	return cast.ToBool(r.Get(key))
}

func (r *Record) GetTime(key string) *gtime.Time {
	v := r.Get(key)
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case *gtime.Time:
		return val
	case time.Time:
		return gtime.New(val)
	case string:
		t, err := gtime.StrToTime(val)
		if err != nil {
			return nil
		}
		return t
	default:
		t := cast.ToTime(v)
		if t.IsZero() {
			return nil
		}
		return gtime.New(t)
	}
}

func (r *Record) GetStringSlice(key string) []string {
	return cast.ToStringSlice(r.Get(key))
}

func (r *Record) GetIntSlice(key string) []int {
	return cast.ToIntSlice(r.Get(key))
}

func (r *Record) Has(key string) bool {
	if r.data == nil {
		return false
	}
	_, ok := r.data[key]
	return ok
}

func (r *Record) Remove(key string) {
	if r.data != nil {
		delete(r.data, key)
	}
}

func (r *Record) Keys() []string {
	keys := make([]string, 0, len(r.data))
	for k := range r.data {
		keys = append(keys, k)
	}
	return keys
}

func (r *Record) IsEmpty() bool {
	return len(r.data) == 0
}
