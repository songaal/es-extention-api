package utils

import (
	"regexp"
	"strings"
)

func Pattern(src, pattern string) bool {
	matched, err := regexp.MatchString(pattern, src)
	return err == nil && matched == true
}

func GetArg(key, defaultValue string, arguments []string) (val string) {
	length := len(arguments)
	for i := 0; i < length; i++ {
		arg := arguments[i]
		if strings.HasPrefix(arg, key) {
			val = strings.Replace(arg, key + "=", "", 1)
		}
	}
	if len(val) == 0 {
		val = defaultValue
	}
	return
}

func Contains(s []string, e string) bool {
	if s == nil {
		return false
	}
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func TypeOf(v interface{}) string {
	switch v.(type) {
	case int:
		return "int"
	case float64:
		return "float64"
	case string:
		return "string"
	case bool:
		return "bool"
	case []interface{}:
		return "list"
	case []string:
		return "list"
	case map[string]interface{}:
		return "object"
	case interface{}:
		return "object"
	default:
		return "unknown"
	}
}