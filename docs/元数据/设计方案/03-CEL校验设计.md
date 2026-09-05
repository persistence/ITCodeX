# ITCodeX 元数据模块 - CEL-Go 校验设计

> 版本: v1.2
> 日期: 2026-09-05
> 技术: Google Common Expression Language (CEL)
>
> 需求文档中的校验来自 NocoBase/Joi。本项目**不引入 Joi**，用内置规则 + CEL 覆盖同等语义。
> HTTP 请求参数（path/query/必填分页等）用 GoFrame `v:` 标签，不在 CEL 里做。CEL 只校验**业务记录**。

## 1. 设计目标

- 使用 cel-go 实现统一的校验引擎
- 支持单字段校验（必填、长度、范围、格式等）
- 支持多字段联合校验（跨字段表达式）
- 支持自定义校验函数扩展
- 编译后程序缓存，高性能执行
- 友好的错误信息提示

### 1.1 需求规则 → CEL

| 需求（Joi 侧） | 字段类型 | CEL / 内置规则 | 阶段 |
|----------------|----------|----------------|------|
| 必填 | 通用 | `data.field != null && data.field != ""` | 一 |
| 最小/最大长度、固定长度 | 字符串 | `size(data.field) >= n` | 一 |
| 正则 | 字符串 | `data.field.matches("...")` | 一 |
| 邮箱 / URL / UUID / 手机 | 对应 Interface | 内置 `email()`/`url()`/`uuid()`/`phone()` | 一～二 |
| 大于/小于/最小/最大 | 数字、日期 | `data.field >= n` | 一 |
| 整数倍 | 数字 | `data.field % n == 0` | 二 |
| 精度 | 数字/百分比 | 自定义函数或 option | 二 |
| 关系必填 | 关系字段 | 外键非空；子表单场景暂不校验 | 三 |
| 多字段联合 | 表级 | 任意 CEL，可用 `data`/`oldData`/`action` | 三 |

## 2. CEL 环境配置

### 2.1 基础 CEL 环境

为每个 Collection 创建独立的 CEL 环境，包含字段变量声明：

```go
package validation

import (
    "github.com/google/cel-go/cel"
    "github.com/google/cel-go/common/types"
    "github.com/google/cel-go/common/types/ref"
)

// BuildCollectionEnv 为Collection构建CEL环境
func BuildCollectionEnv(fields map[string]metadata.Field) (*cel.Env, error) {
    opts := []cel.EnvOption{
        // 声明变量：所有字段可通过字段名访问
        // 同时提供 data map 变量，用于动态访问
        cel.Variable("data", cel.DynType),
        cel.Variable("oldData", cel.DynType), // 更新前的数据（更新时可用）
        cel.Variable("action", cel.StringType), // 操作类型: create/update

        // 标准库
        cel.Lib(StringsLib{}),
        cel.Lib(MathLib{}),
        cel.Lib(TimeLib{}),
        cel.Lib(RegexLib{}),

        // 自定义函数
        cel.Function("email",
            cel.MemberOverload("string_email",
                []*cel.Type{cel.StringType}, cel.BoolType,
                cel.FunctionBinding(func(args ...ref.Val) ref.Val {
                    s := args[0].(types.String)
                    return types.Bool(isValidEmail(string(s)))
                })),
        ),
        cel.Function("phone",
            cel.MemberOverload("string_phone",
                []*cel.Type{cel.StringType}, cel.BoolType,
                cel.FunctionBinding(func(args ...ref.Val) ref.Val {
                    s := args[0].(types.String)
                    return types.Bool(isValidPhoneCN(string(s)))
                })),
        ),
        cel.Function("url",
            cel.MemberOverload("string_url",
                []*cel.Type{cel.StringType}, cel.BoolType,
                cel.FunctionBinding(func(args ...ref.Val) ref.Val {
                    s := args[0].(types.String)
                    return types.Bool(isValidURL(string(s)))
                })),
        ),
        cel.Function("uuid",
            cel.MemberOverload("string_uuid",
                []*cel.Type{cel.StringType}, cel.BoolType,
                cel.FunctionBinding(func(args ...ref.Val) ref.Val {
                    s := args[0].(types.String)
                    return types.Bool(isValidUUID(string(s)))
                })),
        ),
        cel.Function("matches",
            cel.MemberOverload("string_matches_string",
                []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
                cel.FunctionBinding(func(args ...ref.Val) ref.Val {
                    s := args[0].(types.String)
                    pattern := args[1].(types.String)
                    matched, _ := regexp.MatchString(string(pattern), string(s))
                    return types.Bool(matched)
                })),
        ),
        cel.Function("len",
            cel.Overload("len_string",
                []*cel.Type{cel.StringType}, cel.IntType,
                cel.FunctionBinding(func(args ...ref.Val) ref.Val {
                    s := args[0].(types.String)
                    return types.Int(len(s))
                })),
            cel.Overload("len_list",
                []*cel.Type{cel.ListType(cel.DynType)}, cel.IntType,
                cel.FunctionBinding(func(args ...ref.Val) ref.Val {
                    list := args[0].(traits.Lister)
                    return types.Int(list.Size().Value().(int64))
                })),
        ),
    }

    // 为每个字段声明类型化变量（提供类型检查支持）
    for name, field := range fields {
        celType := mapFieldTypeToCEL(field.GetDataType())
        if celType != nil {
            opts = append(opts, cel.Variable(name, celType))
        }
    }

    return cel.NewEnv(opts...)
}

// mapFieldTypeToCEL 将字段数据类型映射到CEL类型
func mapFieldTypeToCEL(dataType string) *cel.Type {
    switch dataType {
    case "string", "text", "uuid", "password":
        return cel.StringType
    case "integer", "bigInt":
        return cel.IntType
    case "float", "double", "decimal":
        return cel.DoubleType
    case "boolean":
        return cel.BoolType
    case "date", "datetime", "timestamp":
        return cel.TimestampType
    default:
        return cel.DynType
    }
}
```

### 2.2 内置校验函数库

```go
package validation

// StringsLib 字符串函数库
type StringsLib struct{}

func (StringsLib) CompileOptions() []cel.EnvOption {
    return []cel.EnvOption{
        cel.Function("startsWith",
            cel.MemberOverload("string_startsWith_string",
                []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
                cel.FunctionBinding(stringStartsWith)),
        ),
        cel.Function("endsWith",
            cel.MemberOverload("string_endsWith_string",
                []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
                cel.FunctionBinding(stringEndsWith)),
        ),
        cel.Function("contains",
            cel.MemberOverload("string_contains_string",
                []*cel.Type{cel.StringType, cel.StringType}, cel.BoolType,
                cel.FunctionBinding(stringContains)),
        ),
        cel.Function("trim",
            cel.MemberOverload("string_trim",
                []*cel.Type{cel.StringType}, cel.StringType,
                cel.FunctionBinding(stringTrim)),
        ),
        cel.Function("lower",
            cel.MemberOverload("string_lower",
                []*cel.Type{cel.StringType}, cel.StringType,
                cel.FunctionBinding(stringLower)),
        ),
        cel.Function("upper",
            cel.MemberOverload("string_upper",
                []*cel.Type{cel.StringType}, cel.StringType,
                cel.FunctionBinding(stringUpper)),
        ),
    }
}

func (StringsLib) ProgramOptions() []cel.ProgramOption {
    return nil
}

// MathLib 数学函数库
type MathLib struct{}

func (MathLib) CompileOptions() []cel.EnvOption {
    return []cel.EnvOption{
        cel.Function("abs",
            cel.Overload("abs_int",
                []*cel.Type{cel.IntType}, cel.IntType,
                cel.FunctionBinding(mathAbsInt)),
            cel.Overload("abs_double",
                []*cel.Type{cel.DoubleType}, cel.DoubleType,
                cel.FunctionBinding(mathAbsDouble)),
        ),
        cel.Function("min",
            cel.Overload("min_int",
                []*cel.Type{cel.IntType, cel.IntType}, cel.IntType,
                cel.FunctionBinding(mathMinInt)),
            cel.Overload("min_double",
                []*cel.Type{cel.DoubleType, cel.DoubleType}, cel.DoubleType,
                cel.FunctionBinding(mathMinDouble)),
        ),
        cel.Function("max",
            cel.Overload("max_int",
                []*cel.Type{cel.IntType, cel.IntType}, cel.IntType,
                cel.FunctionBinding(mathMaxInt)),
            cel.Overload("max_double",
                []*cel.Type{cel.DoubleType, cel.DoubleType}, cel.DoubleType,
                cel.FunctionBinding(mathMaxDouble)),
        ),
    }
}

func (MathLib) ProgramOptions() []cel.ProgramOption {
    return nil
}

// RegexLib 正则表达式库
type RegexLib struct{}

func (RegexLib) CompileOptions() []cel.EnvOption {
    return []cel.EnvOption{
        // matches 函数已在上面定义
    }
}

func (RegexLib) ProgramOptions() []cel.ProgramOption {
    return nil
}

// TimeLib 时间函数库
type TimeLib struct{}

func (TimeLib) CompileOptions() []cel.EnvOption {
    return []cel.EnvOption{
        cel.Function("now",
            cel.Overload("now",
                []*cel.Type{}, cel.TimestampType,
                cel.FunctionBinding(timeNow)),
        ),
        cel.Function("today",
            cel.Overload("today",
                []*cel.Type{}, cel.TimestampType,
                cel.FunctionBinding(timeToday)),
        ),
        cel.Function("date",
            cel.MemberOverload("string_date",
                []*cel.Type{cel.StringType}, cel.TimestampType,
                cel.FunctionBinding(stringToDate)),
        ),
        cel.Function("format",
            cel.MemberOverload("timestamp_format_string",
                []*cel.Type{cel.TimestampType, cel.StringType}, cel.StringType,
                cel.FunctionBinding(timestampFormat)),
        ),
    }
}

func (TimeLib) ProgramOptions() []cel.ProgramOption {
    return nil
}
```

## 3. 单字段校验规则配置

### 3.1 通用配置结构

```go
package validation

// FieldValidationConfig 字段校验配置（存储在字段options中）
type FieldValidationConfig struct {
    // ---- 通用规则 ----
    Required  bool   `json:"required,omitempty"`  // 必填
    Nullable  bool   `json:"nullable,omitempty"`  // 允许null
    Unique    bool   `json:"unique,omitempty"`    // 唯一（数据库级）

    // ---- 字符串规则 ----
    MinLength *int   `json:"minLength,omitempty"`
    MaxLength *int   `json:"maxLength,omitempty"`
    Length    *int   `json:"length,omitempty"`
    Pattern   string `json:"pattern,omitempty"`   // 正则表达式

    // ---- 数字规则 ----
    Min          *float64 `json:"min,omitempty"`
    Max          *float64 `json:"max,omitempty"`
    ExclusiveMin *float64 `json:"exclusiveMin,omitempty"`
    ExclusiveMax *float64 `json:"exclusiveMax,omitempty"`
    MultipleOf   *float64 `json:"multipleOf,omitempty"`
    Integer      bool     `json:"integer,omitempty"`

    // ---- 格式规则 ----
    Format string `json:"format,omitempty"` // email/phone/url/uuid/color

    // ---- 自定义CEL规则 ----
    Rules []CELRule `json:"rules,omitempty"`
}

// CELRule 单字段CEL校验规则
type CELRule struct {
    Name         string `json:"name"`
    Expression   string `json:"expression"`
    ErrorMessage string `json:"errorMessage"`
}
```

### 3.2 规则到 CEL 表达式映射

系统根据配置自动生成 CEL 表达式：

```go
package validation

// BuildFieldRules 从字段配置构建CEL校验程序
func BuildFieldRules(env *cel.Env, fieldName string, config *FieldValidationConfig) ([]CompiledRule, error) {
    var rules []CompiledRule

    fieldAccess := fieldName // 类型化访问
    // 同时支持 data[fieldName] 动态访问

    // Required: 字段不为null且不为空串/空数组
    if config.Required {
        expr := fmt.Sprintf(`%s != null && (type(%s) != string || %s != "") && (type(%s) != list || size(%s) > 0)`,
            fieldAccess, fieldAccess, fieldAccess, fieldAccess, fieldAccess)
        program, err := compileRule(env, fieldName+":required", expr, config.Messages.Required)
        if err != nil {
            return nil, err
        }
        rules = append(rules, program)
    }

    // MinLength
    if config.MinLength != nil {
        expr := fmt.Sprintf(`%s == null || len(%s) >= %d`, fieldAccess, fieldAccess, *config.MinLength)
        msg := fmt.Sprintf("长度不能小于%d", *config.MinLength)
        program, err := compileRule(env, fieldName+":minLength", expr, msg)
        if err != nil {
            return nil, err
        }
        rules = append(rules, program)
    }

    // MaxLength
    if config.MaxLength != nil {
        expr := fmt.Sprintf(`%s == null || len(%s) <= %d`, fieldAccess, fieldAccess, *config.MaxLength)
        msg := fmt.Sprintf("长度不能大于%d", *config.MaxLength)
        program, err := compileRule(env, fieldName+":maxLength", expr, msg)
        if err != nil {
            return nil, err
        }
        rules = append(rules, program)
    }

    // Pattern
    if config.Pattern != "" {
        escapedPattern := strings.ReplaceAll(config.Pattern, `'`, `\'`)
        expr := fmt.Sprintf(`%s == null || %s.matches('%s')`, fieldAccess, fieldAccess, escapedPattern)
        msg := "格式不正确"
        program, err := compileRule(env, fieldName+":pattern", expr, msg)
        if err != nil {
            return nil, err
        }
        rules = append(rules, program)
    }

    // Format: email
    if config.Format == "email" {
        expr := fmt.Sprintf(`%s == null || %s.email()`, fieldAccess, fieldAccess)
        msg := "请输入有效的邮箱地址"
        program, err := compileRule(env, fieldName+":email", expr, msg)
        if err != nil {
            return nil, err
        }
        rules = append(rules, program)
    }

    // Format: phone (中国大陆)
    if config.Format == "phone" {
        expr := fmt.Sprintf(`%s == null || %s.phone()`, fieldAccess, fieldAccess)
        msg := "请输入有效的手机号码"
        program, err := compileRule(env, fieldName+":phone", expr, msg)
        if err != nil {
            return nil, err
        }
        rules = append(rules, program)
    }

    // Format: url
    if config.Format == "url" {
        expr := fmt.Sprintf(`%s == null || %s.url()`, fieldAccess, fieldAccess)
        msg := "请输入有效的URL地址"
        program, err := compileRule(env, fieldName+":url", expr, msg)
        if err != nil {
            return nil, err
        }
        rules = append(rules, program)
    }

    // Format: uuid
    if config.Format == "uuid" {
        expr := fmt.Sprintf(`%s == null || %s.uuid()`, fieldAccess, fieldAccess)
        msg := "请输入有效的UUID"
        program, err := compileRule(env, fieldName+":uuid", expr, msg)
        if err != nil {
            return nil, err
        }
        rules = append(rules, program)
    }

    // Min (数字)
    if config.Min != nil {
        expr := fmt.Sprintf(`%s == null || %s >= %v`, fieldAccess, fieldAccess, *config.Min)
        msg := fmt.Sprintf("值不能小于%v", *config.Min)
        program, err := compileRule(env, fieldName+":min", expr, msg)
        if err != nil {
            return nil, err
        }
        rules = append(rules, program)
    }

    // Max (数字)
    if config.Max != nil {
        expr := fmt.Sprintf(`%s == null || %s <= %v`, fieldAccess, fieldAccess, *config.Max)
        msg := fmt.Sprintf("值不能大于%v", *config.Max)
        program, err := compileRule(env, fieldName+":max", expr, msg)
        if err != nil {
            return nil, err
        }
        rules = append(rules, program)
    }

    // Integer
    if config.Integer {
        expr := fmt.Sprintf(`%s == null || type(%s) == int`, fieldAccess, fieldAccess)
        msg := "请输入整数"
        program, err := compileRule(env, fieldName+":integer", expr, msg)
        if err != nil {
            return nil, err
        }
        rules = append(rules, program)
    }

    // 自定义CEL规则
    for _, rule := range config.Rules {
        program, err := compileRule(env, fieldName+":"+rule.Name, rule.Expression, rule.ErrorMessage)
        if err != nil {
            return nil, fmt.Errorf("规则 %s 编译失败: %w", rule.Name, err)
        }
        rules = append(rules, program)
    }

    return rules, nil
}

type CompiledRule struct {
    Name         string
    Program      cel.Program
    ErrorMessage string
}

func compileRule(env *cel.Env, name, expr, errMsg string) (CompiledRule, error) {
    parsed, issues := env.Parse(expr)
    if issues != nil && issues.Err() != nil {
        return CompiledRule{}, fmt.Errorf("解析错误: %w", issues.Err())
    }

    checked, issues := env.Check(parsed)
    if issues != nil && issues.Err() != nil {
        return CompiledRule{}, fmt.Errorf("类型检查错误: %w", issues.Err())
    }

    program, err := env.Program(checked)
    if err != nil {
        return CompiledRule{}, err
    }

    return CompiledRule{
        Name:         name,
        Program:      program,
        ErrorMessage: errMsg,
    }, nil
}
```

## 4. 多字段联合校验

### 4.1 表级联合校验配置

```go
package validation

// TableValidationConfig 表级校验配置（存储在collection options中）
type TableValidationConfig struct {
    // 联合唯一约束
    UniqueConstraints []UniqueConstraint `json:"uniqueConstraints,omitempty"`

    // 自定义CEL规则（多字段）
    Rules []TableCELRule `json:"rules,omitempty"`
}

type UniqueConstraint struct {
    Name   string   `json:"name"`
    Fields []string `json:"fields"`
}

type TableCELRule struct {
    Name         string   `json:"name"`
    Expression   string   `json:"expression"`
    ErrorMessage string   `json:"errorMessage"`
    Dependencies []string `json:"dependencies"` // 依赖字段，变化时触发
}
```

### 4.2 联合校验示例

```go
// 示例1：开始日期必须早于结束日期
// Expression: data.startDate == null || data.endDate == null || data.startDate <= data.endDate
// ErrorMessage: "开始日期不能晚于结束日期"

// 示例2：金额大于10000时必须填写备注
// Expression: data.amount <= 10000 || (data.remark != null && data.remark != "")
// ErrorMessage: "金额大于10000时必须填写备注"

// 示例3：密码和确认密码必须一致
// Expression: data.password == data.confirmPassword
// ErrorMessage: "两次输入的密码不一致"

// 示例4：状态为已完成时进度必须为100%
// Expression: data.status != 'completed' || data.progress == 100
// ErrorMessage: "已完成状态的进度必须为100%"

// 示例5：邮箱或手机号至少填一个
// Expression: (data.email != null && data.email != "") || (data.phone != null && data.phone != "")
// ErrorMessage: "邮箱和手机号至少填写一个"
```

### 4.3 校验执行流程

```
1. 接收创建/更新请求
2. 数据预处理（类型转换、默认值填充）
3. 单字段校验
   ├─ 遍历所有字段
   ├─ 执行该字段的所有编译后CEL规则
   └─ 收集字段级错误
4. 多字段联合校验
   ├─ 执行表级联合唯一约束检查（数据库查询）
   ├─ 执行表级CEL规则
   └─ 收集表级错误
5. 触发 beforeValidate 事件（Yaegi脚本可介入）
6. 如果有错误，返回 422 Unprocessable Entity
7. 校验通过，继续执行业务逻辑
```

```go
package validation

// ValidateData 执行完整校验流程
func (v *Validator) ValidateData(ctx context.Context, collection *metadata.Collection, data map[string]interface{}, oldData map[string]interface{}, action string) (*ValidationResult, error) {
    result := &ValidationResult{
        FieldErrors:  make(map[string][]string),
        TableErrors:  []string{},
    }

    // 构建激活变量
    activation := map[string]interface{}{
        "data":    data,
        "oldData": oldData,
        "action":  action,
    }

    // 为类型化字段直接赋值
    for name, value := range data {
        activation[name] = value
    }

    // 1. 单字段校验
    for fieldName, field := range collection.GetFields() {
        fieldRules := v.getFieldRules(collection.Name(), fieldName)
        if fieldRules == nil {
            continue
        }

        value := data[fieldName]
        // 如果字段不在data中（部分更新），跳过除required外的规则
        _, exists := data[fieldName]
        for _, rule := range fieldRules {
            // 非required规则，如果字段未提供且无值，跳过（支持部分更新）
            if !exists && !strings.HasSuffix(rule.Name, ":required") {
                if !field.IsRequired() {
                    continue
                }
            }
            // 如果值为null且不是required规则，跳过
            if value == nil && !strings.HasSuffix(rule.Name, ":required") {
                continue
            }

            out, _, err := rule.Program.Eval(activation)
            if err != nil {
                return nil, fmt.Errorf("规则 %s 执行错误: %w", rule.Name, err)
            }

            // 如果返回bool类型
            if out.Type() == cel.BoolType {
                if !out.Value().(bool) {
                    result.AddFieldError(fieldName, rule.ErrorMessage)
                }
            } else if out.Type() == cel.StringType {
                // 如果返回string类型，非空表示错误消息
                msg := out.Value().(string)
                if msg != "" {
                    result.AddFieldError(fieldName, msg)
                }
            }
        }
    }

    // 2. 表级联合校验
    tableRules := v.getTableRules(collection.Name())
    for _, rule := range tableRules {
        out, _, err := rule.Program.Eval(activation)
        if err != nil {
            return nil, fmt.Errorf("表规则 %s 执行错误: %w", rule.Name, err)
        }

        if out.Type() == cel.BoolType {
            if !out.Value().(bool) {
                result.AddTableError(rule.ErrorMessage)
            }
        } else if out.Type() == cel.StringType {
            msg := out.Value().(string)
            if msg != "" {
                result.AddTableError(msg)
            }
        }
    }

    // 3. 唯一约束校验（数据库查询）
    // ... 查询数据库检查联合唯一

    return result, nil
}

type ValidationResult struct {
    FieldErrors  map[string][]string `json:"fieldErrors"`
    TableErrors  []string            `json:"tableErrors"`
}

func (r *ValidationResult) HasErrors() bool {
    return len(r.FieldErrors) > 0 || len(r.TableErrors) > 0
}

func (r *ValidationResult) AddFieldError(field, msg string) {
    r.FieldErrors[field] = append(r.FieldErrors[field], msg)
}

func (r *ValidationResult) AddTableError(msg string) {
    r.TableErrors = append(r.TableErrors, msg)
}
```

## 5. 校验缓存机制

```go
package validation

import (
    "sync"

    "github.com/google/cel-go/cel"
)

// Validator 校验器（缓存编译结果）
type Validator struct {
    mu sync.RWMutex

    // Collection CEL环境缓存: collectionName -> *cel.Env
    envs map[string]*cel.Env

    // 字段规则缓存: collectionName:fieldName -> []CompiledRule
    fieldRules map[string][]CompiledRule

    // 表级规则缓存: collectionName -> []CompiledRule
    tableRules map[string][]CompiledRule
}

func NewValidator() *Validator {
    return &Validator{
        envs:        make(map[string]*cel.Env),
        fieldRules:  make(map[string][]CompiledRule),
        tableRules:  make(map[string][]CompiledRule),
    }
}

// InvalidateCollection 当Collection或Field变更时，使缓存失效
func (v *Validator) InvalidateCollection(collectionName string) {
    v.mu.Lock()
    defer v.mu.Unlock()

    delete(v.envs, collectionName)
    for key := range v.fieldRules {
        if strings.HasPrefix(key, collectionName+":") {
            delete(v.fieldRules, key)
        }
    }
    delete(v.tableRules, collectionName)
}

// InvalidateField 当单个字段变更时，使该字段缓存失效
func (v *Validator) InvalidateField(collectionName, fieldName string) {
    v.mu.Lock()
    defer v.mu.Unlock()

    delete(v.envs, collectionName) // 环境也需要重建
    delete(v.fieldRules, collectionName+":"+fieldName)
    delete(v.tableRules, collectionName)
}
```
