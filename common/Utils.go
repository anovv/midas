package common

import (
	"strconv"
	"time"
)

func ToFloat64(v interface{}) float64 {
	if v == nil {
		return 0.0
	}

	switch v.(type) {
	case float64:
		return v.(float64)
	case string:
		vStr := v.(string)
		vF, _ := strconv.ParseFloat(vStr, 64)
		return vF
	default:
		panic("to float64 error.")
	}
}

func MinIntSlice(v []int) (m int) {
	if len(v) > 0 {
		m = v[0]
	}
	for i := 1; i < len(v); i++ {
		if v[i] < m {
			m = v[i]
		}
	}
	return
}

func MaxIntSlice(v []int) (m int) {
	if len(v) > 0 {
		m = v[0]
	}
	for i := 1; i < len(v); i++ {
		if v[i] > m {
			m = v[i]
		}
	}
	return
}

func ContainsInt(a int, list []int) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func ContainsStrPtr(s string, list []*string) bool {
	if list == nil {
		return false
	}
	for _, str := range list {
		if s == *str {
			return true
		}
	}

	return false
}

func GetKeys(m map[interface{}]interface{}) []interface{} {
	keys := make([]interface{}, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	return keys
}

func FloatToString(input_num float64) string {
	// to convert a float number to a string
	return strconv.FormatFloat(input_num, 'f', 6, 64)
}

func TimeFromUnixTimestampFloat(ts float64) time.Time {
	return time.Unix(0, int64(ts)*int64(time.Millisecond))
}