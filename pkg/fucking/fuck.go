package fucking

import "reflect"

// InterfaceSlice turns any slice into a []interface{}
func InterfaceSlice(slice interface{}) []interface{} {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		panic("InterfaceSlice() given a non-slice type")
	}

	slen := s.Len()
	ret := make([]interface{}, slen)

	for i := 0; i < slen; i++ {
		ret[i] = s.Index(i).Interface()
	}

	return ret
}

func IntSlice(slice []interface{}) []int {
	is := make([]int, len(slice))
	for i, iface := range slice {
		is[i] = iface.(int)
	}
	return is
}