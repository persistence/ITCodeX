package yaegiutils

import (
	"encoding/json"
	"reflect"
	"time"

	"itcodex/server/pkg/utils"
)

func SnowflakeID() int64 {
	return utils.NextID()
}

func HashPassword(p string) string {
	return utils.HashPassword(p)
}

func UUID() string {
	return utils.UUID()
}

func NanoID() string {
	return utils.NanoID()
}

func Now() time.Time {
	return time.Now()
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

var Symbols = map[string]map[string]reflect.Value{
	"itcodex/utils/utils": {
		"SnowflakeID":  reflect.ValueOf(SnowflakeID),
		"HashPassword": reflect.ValueOf(HashPassword),
		"UUID":         reflect.ValueOf(UUID),
		"NanoID":       reflect.ValueOf(NanoID),
		"Now":          reflect.ValueOf(Now),
		"ToJSON":       reflect.ValueOf(ToJSON),
		"FromJSON":     reflect.ValueOf(FromJSON),
	},
}
