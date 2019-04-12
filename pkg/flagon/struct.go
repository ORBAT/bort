package flagon

import (
	"flag"
	"log"
	"reflect"
	"strings"
	"unsafe"
)

func Struct(s interface{}) {
	sVal := reflect.ValueOf(s).Elem()
	sType := sVal.Type()
	for i := 0; i < sType.NumField(); i++ {
		typeField := sType.Field(i)
		name := typeField.Name
		name = strings.ToLower(name[:1]) + name[1:]
		valField := sVal.Field(i)
		valPtr := unsafe.Pointer(valField.Addr().Pointer())
		fieldDescr := typeField.Tag.Get("usage")
		if fieldDescr == "" {
			fieldDescr = typeField.Name
		}

		switch kind := typeField.Type.Kind(); kind {
		case reflect.Int:
			flag.IntVar((*int)(valPtr), name, *(*int)(valPtr), fieldDescr)
		case reflect.Float64:
			flag.Float64Var((*float64)(valPtr), name, *(*float64)(valPtr), fieldDescr)
		case reflect.Bool:
			flag.BoolVar((*bool)(valPtr), name, *(*bool)(valPtr), fieldDescr)
		case reflect.Ptr:
			if typeField.Type.Elem().Kind() == reflect.Struct {
				Struct(valField.Addr().Interface())
			}
		case reflect.Struct:
			Struct(valField.Addr().Interface())
		default:
			log.Fatalf("I can't deal with fields of kind %s", kind.String())
		}
	}
}

