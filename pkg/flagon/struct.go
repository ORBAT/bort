package flagon

import (
	"flag"
	"log"
	"reflect"
	"strings"
	"unicode"
	"unsafe"
)

func lowerFirst(s string) string {
	if len(s) < 2 {
		return strings.ToLower(s)
	}
	if unicode.IsUpper(rune(s[1])) {
		return s
	}

	var sb strings.Builder
	sb.WriteString(strings.ToLower(s[:1]))
	sb.WriteString(s[1:])
	return sb.String()
}

func addDot(prefixes []string) string {
	var sb strings.Builder
	for _, pfx := range prefixes {
		sb.WriteString(pfx)
		sb.WriteRune('.')
	}
	return lowerFirst(sb.String())
}
// Struct creates command line flags (using the standard lib flag package) that set fields of s,
// prefixed with prefixes separated by dots. The flag name is the field name but with the first
// letter lowercased (unless the second letter is uppercase as well) and in case of struct fields
// prefixed with the field name. For example,
// 	MyField bool
// would give the flag
// 	-myField
//
// and the field CPU.Bla in
//  type CPU struct{ Bla bool }
// 	type Conf { CPU; MyField bool }
// would have the flag
// 	-CPU.bla
//
// Current contents of s are used as default values, and the struct tag "usage" can be used to
// provide usage texts for flags. Struct can be called multiple times before flag.Parse() is called.
//
// s will be mutated when flag.Parse() is called.
//
// Note that Struct doesn't support UTF8/Unicode field names
func Struct(s interface{}, prefixes ...string) {
	sVal := reflect.ValueOf(s).Elem()
	sType := sVal.Type()
	for i := 0; i < sType.NumField(); i++ {
		typeField := sType.Field(i)
		name := typeField.Name
		name = addDot(prefixes) + lowerFirst(name)
		valField := sVal.Field(i)
		valPtr := unsafe.Pointer(valField.Addr().Pointer())
		usageTag := typeField.Tag.Get("usage")

		switch kind := typeField.Type.Kind(); kind {
		case reflect.Int:
			flag.IntVar((*int)(valPtr), name, *(*int)(valPtr), usageTag)
		case reflect.Float64:
			flag.Float64Var((*float64)(valPtr), name, *(*float64)(valPtr), usageTag)
		case reflect.Bool:
			flag.BoolVar((*bool)(valPtr), name, *(*bool)(valPtr), usageTag)
		case reflect.Ptr:
			if typeField.Type.Elem().Kind() == reflect.Struct {
				Struct(valField.Addr().Interface(), typeField.Name)
			}
		case reflect.Struct:
			Struct(valField.Addr().Interface(), typeField.Name)
		default:
			log.Fatalf("I can't deal with fields of kind %s", kind.String())
		}
	}
}

