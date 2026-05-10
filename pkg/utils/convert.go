package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// ToInt64 converts common scalar types to int64.
func ToInt64(v any) (int64, error) {
	switch n := v.(type) {
	case int:
		return int64(n), nil
	case int8:
		return int64(n), nil
	case int16:
		return int64(n), nil
	case int32:
		return int64(n), nil
	case int64:
		return n, nil
	case uint:
		return int64(n), nil
	case uint8:
		return int64(n), nil
	case uint16:
		return int64(n), nil
	case uint32:
		return int64(n), nil
	case uint64:
		if n > uint64(^uint64(0)>>1) {
			return 0, fmt.Errorf("value out of int64 range: %d", n)
		}
		return int64(n), nil
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
		if err != nil {
			return 0, err
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unsupported int conversion type: %T", v)
	}
}

// ToInt converts to int with overflow checking.
func ToInt(v any) (int, error) {
	n, err := ToInt64(v)
	if err != nil {
		return 0, err
	}
	out := int(n)
	if int64(out) != n {
		return 0, fmt.Errorf("value out of int range: %d", n)
	}
	return out, nil
}

// ToBool converts common scalar types to bool.
func ToBool(v any) (bool, error) {
	switch b := v.(type) {
	case bool:
		return b, nil
	case string:
		return strconv.ParseBool(strings.TrimSpace(b))
	case int, int8, int16, int32, int64:
		n, err := ToInt64(v)
		if err != nil {
			return false, err
		}
		return n != 0, nil
	case uint, uint8, uint16, uint32, uint64:
		n, err := ToInt64(v)
		if err != nil {
			return false, err
		}
		return n != 0, nil
	default:
		return false, fmt.Errorf("unsupported bool conversion type: %T", v)
	}
}
