package ebschedule

import (
	"fmt"

	"github.com/mattn/go-jsonpointer"
)

func getValue[T any](s any, path string, dst **T) (found bool, err error) {
	if !jsonpointer.Has(s, path) {
		return false, nil
	}

	val, err := jsonpointer.Get(s, path)
	if err != nil {
		return false, fmt.Errorf("jsonpointer.Get: %w", err)
	}

	*dst = nil
	v, ok := val.(T)
	if !ok {
		return true, fmt.Errorf("type mismatch: val=%T, dst=%T", val, *dst)
	}

	*dst = &v
	return true, nil
}

func removeValue[T any](s T, path string) (ret T, removed bool, err error) {
	if !jsonpointer.Has(s, path) {
		return s, false, nil
	}

	v, err := jsonpointer.Remove(s, path)
	if err != nil {
		return ret, false, err
	}

	return v.(T), true, nil
}

func setValue(s any, path string, v any) error {
	return jsonpointer.Set(s, path, v)
}
