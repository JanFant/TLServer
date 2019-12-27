package data

import (
	"fmt"
	"reflect"
)

func ParserInterface(in interface{}) (contx map[string]string) {
	contx = make(map[string]string)
	v := reflect.ValueOf(in)
	if v.Kind() == reflect.Map {
		for _, key := range v.MapKeys() {
			strct := v.MapIndex(key)
			contx[fmt.Sprint(key.Interface())] = fmt.Sprint(strct.Interface())
		}
	}
	return
}
