package common

import "errors"

func ValidateNil(data interface{}, msg string) error {
	if data == nil {
		return errors.New(msg + ` must be specified`)
	}
	return nil
}
func ByteSliceEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	if (a == nil) != (b == nil) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
