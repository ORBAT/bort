package fucking

import "reflect"

// InterfaceSlice turns any slice into a []interface{}
func InterfaceSlice(slice interface{}) []interface{} {
	switch t := slice.(type) {
	case []int:
		slen := len(t)
		ret := make([]interface{}, 0, slen)
		for _, num := range t {
			ret = append(ret, num)
		}
		return ret
	default:
	}

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
