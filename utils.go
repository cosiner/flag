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
	k := val.Kind()
	if k == reflect.Slice {
		return val.Type().Elem().Kind()
	}
	return k
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
		switch k := sliceElemKind(refval); k {
		case reflect.String:
			return vals, nil
		case reflect.Bool:
			bs, err := convertToBools(vals)
			return bs, err
		default:
			if isKindNumber(k) {
				fs, err := convertToFloats(vals)
				return fs, err
			}
		}
	}
	return nil, errors.New("unsupported kind")
}

func convertNumberSlice(val interface{}) []float64 {
	var fs []float64
	switch vals := val.(type) {
	case []int:
		fs = make([]float64, len(vals))
		for i, v := range vals {
			fs[i] = float64(v)
		}
	case []int8:
		fs = make([]float64, len(vals))
		for i, v := range vals {
			fs[i] = float64(v)
		}
	case []int16:
		fs = make([]float64, len(vals))
		for i, v := range vals {
			fs[i] = float64(v)
		}
	case []int32:
		fs = make([]float64, len(vals))
		for i, v := range vals {
			fs[i] = float64(v)
		}
	case []int64:
		fs = make([]float64, len(vals))
		for i, v := range vals {
			fs[i] = float64(v)
		}
	case []uint:
		fs = make([]float64, len(vals))
		for i, v := range vals {
			fs[i] = float64(v)
		}
	case []uint8:
		fs = make([]float64, len(vals))
		for i, v := range vals {
			fs[i] = float64(v)
		}
	case []uint16:
		fs = make([]float64, len(vals))
		for i, v := range vals {
			fs[i] = float64(v)
		}
	case []uint32:
		fs = make([]float64, len(vals))
		for i, v := range vals {
			fs[i] = float64(v)
		}
	case []uint64:
		fs = make([]float64, len(vals))
		for i, v := range vals {
			fs[i] = float64(v)
		}
	case []float32:
		fs = make([]float64, len(vals))
		for i, v := range vals {
			fs[i] = float64(v)
		}
	case []float64:
		fs = vals
	}
	return fs
}

func convertToFloats(vals []string) ([]float64, error) {
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

func convertToBools(vals []string) ([]bool, error) {
	bs := make([]bool, 0, len(vals))
	for _, v := range vals {
		f, err := parseBool(v)
		if err != nil {
			return nil, err
		}
		bs = append(bs, f)
	}
	return bs, nil
}

func parseSelectsString(val, valsep string, ptr interface{}) (interface{}, error) {
	if val == "" {
		return nil, nil
	}

	refval := reflect.ValueOf(ptr).Elem()
	vals := splitAndTrimSpace(val, valsep)
	k := sliceElemKind(refval)
	switch {
	case k == reflect.String:
		return vals, nil
	case isKindNumber(k):
		return convertToFloats(vals)
	}
	return nil, fmt.Errorf("doesn't support select: %s", k.String())
}

func parseSelectsValue(ptr interface{}, val interface{}) (interface{}, error) {
	if val == nil {
		return nil, nil
	}

	refval := reflect.ValueOf(ptr).Elem()
	k := sliceElemKind(refval)
	if isKindNumber(k) {
		fs := convertNumberSlice(val)
		if len(fs) != 0 {
			return fs, nil
		}
	} else if k == reflect.String {
		if vals, ok := val.([]string); ok && len(vals) != 0 {
			return vals, nil
		}
	}
	return nil, errors.New("invalid selects")
}

func valSep(sep string) string {
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

func checkSelects(k reflect.Kind, selects interface{}, val string, flt float64) bool {
	var valid bool
	switch {
	case isKindNumber(k):
		vals, _ := selects.([]float64)
		for _, v := range vals {
			valid = valid || flt == v
			if valid {
				break
			}
		}
	case k == reflect.String:
		vals, _ := selects.([]string)
		for _, v := range vals {
			valid = valid || v == val
			if valid {
				break
			}
		}
	}
	return valid
}

func applyValToPtr(names string, ptr interface{}, val string, selects interface{}) error {
	var err error
	if isBoolPtr(ptr) {
		val, err = convertBool(val)
		if err != nil {
			return fmt.Errorf("%s: %s", names, err.Error())
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
	if err != nil {
		return fmt.Errorf("%s: %s", names, err.Error())
	}
	if selects != nil {
		refval := reflect.ValueOf(ptr).Elem()
		k := sliceElemKind(refval)
		if !checkSelects(k, selects, val, flt) {
			return fmt.Errorf("%s: invalid value %s of %v", names, val, selects)
		}
	}
	return err
}
