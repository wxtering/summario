package utils

import "errors"

// AsType extracts an error of type T from err chain using errors.As.
func AsType[T error](err error) (T, bool) {
	var target T
	if errors.As(err, &target) {
		return target, true
	}
	return target, false
}
