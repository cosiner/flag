package flag

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

func isKindNumber(k reflect.Kind) bool {
	switch k {
	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Float32,
		reflect.Float64:
		return true
	}
	return false
}

func isKindCompatible(k1, k2 reflect.Kind) bool {
	switch k1 {
	case reflect.Bool, reflect.String:
		return k2 == k1
	}
	return isKindNumber(k1) && isKindNumber(k2)
}

func sliceElemKind(val reflect.Value) reflect.Kind {
	return val.Type().Elem().Kind()
}

func isBoolPtr(ptr interface{}) bool {
	switch ptr.(type) {
	case *bool, *[]bool:
		return true
	}
	return false
}

func isSlicePtr(ptr interface{}) bool {
	refval := reflect.ValueOf(ptr)
	return refval.Kind() == reflect.Ptr && refval.Elem().Kind() == reflect.Slice
}

func splitAndTrimSpace(s, sep string) []string {
	secs := strings.Split(s, sep)
	for i := range secs {
		secs[i] = strings.TrimSpace(secs[i])
	}
	return secs
}

func unexportedName(name string) string {
	for _, r := range name {
		if unicode.IsUpper(r) {
			bs := []rune(name)
			for i, r := range bs {
				if unicode.IsUpper(r) {
					bs[i] = unicode.ToLower(r)
				} else {
					break
				}
			}
			return string(bs)
		}
	}
	return name
}

func convertBool(val string) (string, error) {
	switch strings.ToLower(val) {
	case "true", "t", "yes", "y", "1":
		return "true", nil
	case "false", "f", "no", "n", "0":
		return "false", nil
	}
	return "", fmt.Errorf("illegal boolean value: %s", val)
}

func parseBool(val string) (bool, error) {
	val, err := convertBool(val)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(val)
}

func parseDefault(val, valsep string, ptr interface{}) (interface{}, error) {
	if val == "" {
		return nil, nil
	}
	refval := reflect.ValueOf(ptr).Elem()
	switch refval.Kind() {
	case reflect.String:
		return val, nil
	case reflect.Bool:
		b, err := parseBool(val)
		return b, err
	default:
		if isKindNumber(refval.Kind()) {
			return strconv.ParseFloat(val, 64)
		}
	case reflect.Slice:
		vals := splitAndTrimSpace(val, valsep)
		switch k := refval.Type().Elem().Kind(); k {
		case reflect.String:
			return vals, nil
		case reflect.Bool:
			bs := make([]bool, 0, len(vals))
			for _, v := range vals {
				b, err := parseBool(v)
				if err != nil {
					return nil, err
				}
				bs = append(bs, b)
			}
			return bs, nil
		default:
			if isKindNumber(k) {
				fs := make([]float64, 0, len(vals))
				for _, v := range vals {
					f, err := strconv.ParseFloat(v, 64)
					if err != nil {
						return nil, err
					}
					fs = append(fs, f)
				}
				return fs, nil
			}
		}
	}
	return nil, errors.New("unsupported kind")
}

func envValSep(sep string) string {
	if sep == "" {
		return ","
	}
	return sep
}

func typeName(ptr interface{}) string {
	switch ptr.(type) {
	case *int:
		return "int"
	case *int8:
		return "int8"
	case *int16:
		return "int16"
	case *int32:
		return "int32"
	case *int64:
		return "int64"
	case *[]int:
		return "[]int"
	case *[]int8:
		return "[]int8"
	case *[]int16:
		return "[]int16"
	case *[]int32:
		return "[]int32"
	case *[]int64:
		return "[]int64"
	case *uint:
		return "uint"
	case *uint8:
		return "uint8"
	case *uint16:
		return "uint16"
	case *uint32:
		return "uint32"
	case *uint64:
		return "uint64"
	case *[]uint:
		return "[]uint"
	case *[]uint8:
		return "[]uint8"
	case *[]uint16:
		return "[]uint16"
	case *[]uint32:
		return "[]uint32"
	case *[]uint64:
		return "[]uint64"
	case *float32:
		return "float32"
	case *float64:
		return "float64"
	case *[]float32:
		return "[]float32"
	case *[]float64:
		return "[]float64"
	case *string:
		return "string"
	case *[]string:
		return "[]string"
	case *bool:
		return "bool"
	case *[]bool:
		return "[]bool"
	}
	return ""
}

func applyValToPtr(ptr interface{}, val string) error {
	var err error
	if isBoolPtr(ptr) {
		val, err = convertBool(val)
		if err != nil {
			return err
		}
	}

	flt, ferr := strconv.ParseFloat(val, 64)
	bl, berr := strconv.ParseBool(val)
	switch v := ptr.(type) {
	case *int:
		*v, err = int(flt), ferr
	case *int8:
		*v, err = int8(flt), ferr
	case *int16:
		*v, err = int16(flt), ferr
	case *int32:
		*v, err = int32(flt), ferr
	case *int64:
		*v, err = int64(flt), ferr
	case *uint:
		*v, err = uint(flt), ferr
	case *uint8:
		*v, err = uint8(flt), ferr
	case *uint16:
		*v, err = uint16(flt), ferr
	case *uint32:
		*v, err = uint32(flt), ferr
	case *uint64:
		*v, err = uint64(flt), ferr
	case *float32:
		*v, err = float32(flt), ferr
	case *float64:
		*v, err = float64(flt), ferr
	case *[]int:
		*v, err = append(*v, int(flt)), ferr
	case *[]int8:
		*v, err = append(*v, int8(flt)), ferr
	case *[]int16:
		*v, err = append(*v, int16(flt)), ferr
	case *[]int32:
		*v, err = append(*v, int32(flt)), ferr
	case *[]int64:
		*v, err = append(*v, int64(flt)), ferr
	case *[]uint:
		*v, err = append(*v, uint(flt)), ferr
	case *[]uint8:
		*v, err = append(*v, uint8(flt)), ferr
	case *[]uint16:
		*v, err = append(*v, uint16(flt)), ferr
	case *[]uint32:
		*v, err = append(*v, uint32(flt)), ferr
	case *[]uint64:
		*v, err = append(*v, uint64(flt)), ferr
	case *[]float32:
		*v, err = append(*v, float32(flt)), ferr
	case *[]float64:
		*v, err = append(*v, float64(flt)), ferr
	case *string:
		*v = val
	case *[]string:
		*v = append(*v, val)
	case *bool:
		*v, err = bl, berr
	case *[]bool:
		*v, err = append(*v, bl), berr
	default:
		err = errors.New("unsupported type")
	}
	return err
}
