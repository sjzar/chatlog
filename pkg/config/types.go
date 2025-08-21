package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// DecodeStringToMap returns a DecodeHookFunc that converts a string to a map[string]string.
// The string is expected to be a comma-separated list of key-value pairs, where the key and value
// are separated by an equal sign.
func DecodeStringToMap() mapstructure.DecodeHookFunc {
	return func(f reflect.Kind, t reflect.Kind, data interface{}) (interface{}, error) {
		// check if field is a string and target is a map
		if f != reflect.String || t != reflect.Map {
			return data, nil
		}
		// check if target is map[string]string
		if t != reflect.TypeOf(map[string]string{}).Kind() {
			return data, nil
		}

		raw := data.(string)
		if raw == "" {
			return map[string]string{}, nil
		}
		// parse raw string as key1=value1,key2=value2
		const pairSep = ","
		const valueSep = "="
		pairs := strings.Split(raw, pairSep)
		m := make(map[string]string, len(pairs))
		for _, pair := range pairs {
			key, value, found := strings.Cut(pair, valueSep)
			if !found {
				return nil, fmt.Errorf("invalid key-value pair: %s", pair)
			}
			m[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}

		return m, nil
	}
}

// StringToSliceWithBracketHookFunc returns a DecodeHookFunc that converts a string to a slice of strings.
// Useful when configuration values are provided as JSON arrays in string form, but need to be parsed into slices.
// The string is expected to be a JSON array.
// If the string is empty, an empty slice is returned.
// If the string cannot be parsed as a JSON array, the original data is returned unchanged.
func StringToSliceWithBracketHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Kind, t reflect.Kind, data interface{}) (interface{}, error) {
		if f != reflect.String || t != reflect.Slice {
			return data, nil
		}

		raw := data.(string)
		if raw == "" {
			return []string{}, nil
		}
		var result any
		err := json.Unmarshal([]byte(raw), &result)
		if err != nil {
			return data, nil
		}

		// Verify that the result matches the target (slice)
		if reflect.TypeOf(result).Kind() != t {
			return data, nil
		}
		return result, nil
	}
}

// StringToStructHookFunc returns a DecodeHookFunc that converts a string to a struct.
// Useful for parsing configuration values that are provided as JSON strings but need to be converted to sturcts.
// The string is expected to be a JSON object that can be unmarshaled into the target struct.
// If the string is empty, a new instance of the target struct is returned.
// If the string cannot be parsed as a JSON object, the original data is returned unchanged.
func StringToStructHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String ||
			(t.Kind() != reflect.Struct && !(t.Kind() == reflect.Pointer && t.Elem().Kind() == reflect.Struct)) {
			return data, nil
		}
		raw := data.(string)
		var val reflect.Value
		// Struct or the pointer to a struct
		if t.Kind() == reflect.Struct {
			val = reflect.New(t)
		} else {
			val = reflect.New(t.Elem())
		}

		if raw == "" {
			return val, nil
		}
		var m map[string]interface{}
		err := json.Unmarshal([]byte(raw), &m)
		if err != nil {
			return data, nil
		}
		return m, nil
	}
}

// CompositeDecodeHook 组合所有解码钩子
func CompositeDecodeHook() mapstructure.DecodeHookFunc {
	return mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		DecodeStringToMap(),
		StringToStructHookFunc(),
		StringToSliceWithBracketHookFunc(),
	)
}

func decoderConfig() viper.DecoderConfigOption {
	return viper.DecodeHook(CompositeDecodeHook())
}
