package helpers

import (
	"reflect"
)

func SearchInArray(array interface{}, predicate func(interface{}) bool) int {
	if array == nil {
		return -1
	}

	value := reflect.ValueOf(array)
	if value.IsNil() {
		return -1
	}

	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		n := value.Len()
		for i := 0; i < n; i++ {
			itemValue := value.Index(i)
			item := itemValue.Interface()
			if predicate(item) {
				return i
			}
		}
		return -1

	default:
		panic("This function should only called for slices or arrays")
	}
}
func SearchInArrayI(array interface{}, predicate func(int) bool) int {
	if array == nil {
		return -1
	}

	value := reflect.ValueOf(array)
	if value.IsNil() {
		return -1
	}

	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		n := value.Len()
		for i := 0; i < n; i++ {
			if predicate(i) {
				return i
			}
		}
		return -1

	default:
		panic("This function should only called for slices or arrays")
	}
}

func FilterArray(array interface{}, predicate func(interface{}) bool) interface{} {
	value := reflect.ValueOf(array)
	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		if value.IsNil() {
			return array
		}

		result := reflect.New(value.Type())
		n := value.Len()
		for i := 0; i < n; i++ {
			itemValue := value.Index(i)
			item := itemValue.Interface()
			if predicate(item) {
				result = reflect.Append(result, itemValue)
			}
		}
		return result.Interface()

	default:
		panic("This function should only called for slices or arrays")
	}
}
func FilterArrayI(array interface{}, predicate func(int) bool) interface{} {
	value := reflect.ValueOf(array)
	switch value.Kind() {
	case reflect.Slice, reflect.Array:
		if value.IsNil() {
			return array
		}

		result := reflect.New(value.Type())
		n := value.Len()
		for i := 0; i < n; i++ {
			itemValue := value.Index(i)
			if predicate(i) {
				result = reflect.Append(result, itemValue)
			}
		}
		return result.Interface()

	default:
		panic("This function should only called for slices or arrays")
	}
}
