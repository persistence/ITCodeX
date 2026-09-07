package yaegiutils

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"itcodex/server/pkg/utils"
)

func SnowflakeID() int64 { return utils.NextID() }
func UUID() string       { return utils.UUID() }
func NanoID() string     { return utils.NanoID() }
func Now() time.Time     { return time.Now() }

func HashPassword(p string) string { return utils.HashPassword(p) }
func VerifyPassword(password, hash string) bool {
	return utils.HashPassword(password) == hash
}
func MD5(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}
func SHA256(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func Contains(s, substr string) bool { return strings.Contains(s, substr) }
func HasPrefix(s, prefix string) bool { return strings.HasPrefix(s, prefix) }
func HasSuffix(s, suffix string) bool { return strings.HasSuffix(s, suffix) }
func Trim(s string) string            { return strings.TrimSpace(s) }
func TrimSpace(s string) string       { return strings.TrimSpace(s) }
func ToLower(s string) string         { return strings.ToLower(s) }
func ToUpper(s string) string         { return strings.ToUpper(s) }
func Split(s, sep string) []string    { return strings.Split(s, sep) }
func Join(elems []string, sep string) string {
	return strings.Join(elems, sep)
}
func Replace(s, old, new string, n int) string {
	return strings.Replace(s, old, new, n)
}

func ToJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
func FromJSON(s string) (map[string]any, error) {
	out := map[string]any{}
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, err
	}
	return out, nil
}
func ToMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	out := map[string]any{}
	_ = json.Unmarshal(b, &out)
	return out
}

func ParseTime(s string, layout ...string) (time.Time, error) {
	l := time.RFC3339
	if len(layout) > 0 && layout[0] != "" {
		l = layout[0]
	}
	return time.Parse(l, s)
}
func FormatTime(t time.Time, layout ...string) string {
	l := time.RFC3339
	if len(layout) > 0 && layout[0] != "" {
		l = layout[0]
	}
	return t.Format(l)
}

func LogInfo(v ...any)  { log.Println(append([]any{"[yaegi][info]"}, v...)...) }
func LogWarn(v ...any)  { log.Println(append([]any{"[yaegi][warn]"}, v...)...) }
func LogError(v ...any) { log.Println(append([]any{"[yaegi][error]"}, v...)...) }

func NanoIDLen(n int) string {
	if n <= 0 {
		return utils.NanoID()
	}
	return utils.NanoID()
}

var Symbols = map[string]map[string]reflect.Value{
	"itcodex/utils/utils": {
		"SnowflakeID":    reflect.ValueOf(SnowflakeID),
		"HashPassword":   reflect.ValueOf(HashPassword),
		"VerifyPassword": reflect.ValueOf(VerifyPassword),
		"UUID":           reflect.ValueOf(UUID),
		"NanoID":         reflect.ValueOf(NanoID),
		"Now":            reflect.ValueOf(Now),
		"ToJSON":         reflect.ValueOf(ToJSON),
		"FromJSON":       reflect.ValueOf(FromJSON),
		"ToMap":          reflect.ValueOf(ToMap),
		"Contains":       reflect.ValueOf(Contains),
		"HasPrefix":      reflect.ValueOf(HasPrefix),
		"HasSuffix":      reflect.ValueOf(HasSuffix),
		"Trim":           reflect.ValueOf(Trim),
		"TrimSpace":      reflect.ValueOf(TrimSpace),
		"ToLower":        reflect.ValueOf(ToLower),
		"ToUpper":        reflect.ValueOf(ToUpper),
		"Split":          reflect.ValueOf(Split),
		"Join":           reflect.ValueOf(Join),
		"Replace":        reflect.ValueOf(Replace),
		"ParseTime":      reflect.ValueOf(ParseTime),
		"FormatTime":     reflect.ValueOf(FormatTime),
		"MD5":            reflect.ValueOf(MD5),
		"SHA256":         reflect.ValueOf(SHA256),
		"LogInfo":        reflect.ValueOf(LogInfo),
		"LogWarn":        reflect.ValueOf(LogWarn),
		"LogError":       reflect.ValueOf(LogError),
		"NanoIDLen":      reflect.ValueOf(NanoIDLen),
		"Sprintf":        reflect.ValueOf(fmt.Sprintf),
	},
}
