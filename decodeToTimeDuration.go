package main

import (
	"fmt"
	"reflect"
	"time"
)

func decodeToTimeDuration(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t != reflect.TypeOf(time.Duration(5)) {
		return data, nil
	}

	if f.Kind() != reflect.Int && f.Kind() != reflect.String {
		return nil, fmt.Errorf("wrong type in value for time.Duration")
	}

	if f.Kind() == reflect.Int {
		return time.Duration(time.Duration(data.(int)) * time.Second), nil
	}

	// Convert it by parsing
	return time.ParseDuration(data.(string))
}
