package json

import "reflect"

func GetJSONTag(structType reflect.Type, fieldName string) string {
	field, found := structType.Elem().FieldByName(fieldName)
	if !found {
		return fieldName
	}
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" || jsonTag == "-" {
		return fieldName
	}
	return jsonTag
}
