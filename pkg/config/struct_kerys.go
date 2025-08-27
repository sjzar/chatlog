package config

import (
	"reflect"
	"strings"
)

const sep = "."

// GetStructKeys returns all keys in a nested struct type, taking the name from the tag name or
// the field name.  It handles an additional suffix squashValue like mapstructure does: if
// present on an embedded struct, name components for that embedded struct should not be
// included.  It does not handle maps, does chase pointers, but does not check for loops in
// nesting.
func GetStructKeys(typ reflect.Type, tag, squashValue string) []string {
	return appendStructKeys(typ, tag, ","+squashValue, nil, nil)
}

// appendStructKeys recursively appends to keys all keys of nested struct type typ, taking tag
// and squashValue from GetStructKeys.  prefix holds all components of the path from the typ
// passed to GetStructKeys down to this typ.
func appendStructKeys(typ reflect.Type, tag, squashValue string, prefix []string, keys []string) []string {
	// Dereference any pointers.  This is a finite loop: Go types are well-founded.
	for ; typ.Kind() == reflect.Ptr; typ = typ.Elem() {
	}

	// Handle only struct containers; terminate the recursion on anything else.
	if typ.Kind() != reflect.Struct {
		return append(keys, strings.Join(prefix, sep))
	}

	for i := 0; i < typ.NumField(); i++ {
		fieldType := typ.Field(i)
		var (
			// fieldName is the name to use for the field.
			fieldName string
			// If squash is true, squash the sub-struct no additional accessor.
			squash bool
			ok     bool
		)
		if fieldName, ok = fieldType.Tag.Lookup(tag); ok {
			if strings.HasSuffix(fieldName, squashValue) {
				squash = true
				fieldName = strings.TrimSuffix(fieldName, squashValue)
			}
		} else {
			fieldName = strings.ToLower(fieldType.Name)
		}
		// Update prefix to recurse into this field.
		if !squash {
			prefix = append(prefix, fieldName)
		}
		keys = appendStructKeys(fieldType.Type, tag, squashValue, prefix, keys)
		// Restore prefix.
		if !squash {
			prefix = prefix[:len(prefix)-1]
		}
	}
	return keys
}

// ValidateMissingRequiredKeys returns all keys of value in GetStructKeys format that have an
// additional required tag set but are unset.
func ValidateMissingRequiredKeys(value interface{}, tag, squashValue string) []string {
	return appendStructKeysIfZero(reflect.ValueOf(value), tag, ","+squashValue, "validate", "required", nil, nil)
}

// isScalar returns true if kind is "scalar", i.e. has no Elem().  This
// recites the list from reflect/type.go and may start to give incorrect
// results if new kinds are added to the language.
func isScalar(kind reflect.Kind) bool {
	switch kind {
	case reflect.Array:
	case reflect.Chan:
	case reflect.Map:
	case reflect.Ptr:
	case reflect.Slice:
		return false
	}
	return true
}

func appendStructKeysIfZero(value reflect.Value, tag, squashValue, validateTag, requiredValue string, prefix []string, keys []string) []string {
	// finite loop: Go types are well-founded.
	for value.Kind() == reflect.Ptr {
		if value.IsZero() { // If required, would already have errored out.
			return keys
		}
		value = value.Elem()
	}

	if !isScalar(value.Kind()) {
		// Why use Type().Elem() when reflect.Value provides a perfectly good Elem()
		// method?  The two are *not* the same, e.g. for nil pointers value.Elem() is
		// invalid and has no Kind().  (See https://play.golang.org/p/M3ZV19AZAW0)
		if !isScalar(value.Type().Elem().Kind()) {
			// TODO(ariels): Possible to add, but need to define the semantics.  One
			//     way might be to validate each field according to its type.
			panic("No support for detecting required keys inside " + value.Kind().String() + " of structs")
		}
	}

	// Handle only struct containers; terminate the recursion on anything else.
	if value.Kind() != reflect.Struct {
		return keys
	}

	for i := 0; i < value.NumField(); i++ {
		fieldType := value.Type().Field(i)
		fieldValue := value.Field(i)

		var (
			// fieldName is the name to use for the field.
			fieldName string
			// If squash is true, squash the sub-struct no additional accessor.
			squash bool
			ok     bool
		)
		if fieldName, ok = fieldType.Tag.Lookup(tag); ok {
			if strings.HasSuffix(fieldName, squashValue) {
				squash = true
				fieldName = strings.TrimSuffix(fieldName, squashValue)
			}
		} else {
			fieldName = strings.ToLower(fieldType.Name)
		}

		// Perform any needed validations.
		if validationsString, ok := fieldType.Tag.Lookup(validateTag); ok {
			for _, validation := range strings.Split(validationsString, ",") {
				// Validate "required" field.
				if validation == requiredValue && fieldValue.IsZero() {
					keys = append(keys, strings.Join(append(prefix, fieldName), sep))
				}
			}
		}

		// Update prefix to recurse into this field.
		if !squash {
			prefix = append(prefix, fieldName)
		}
		keys = appendStructKeysIfZero(fieldValue, tag, squashValue, validateTag, requiredValue, prefix, keys)
		// Restore prefix.
		if !squash {
			prefix = prefix[:len(prefix)-1]
		}
	}
	return keys
}
