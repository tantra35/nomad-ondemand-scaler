package nodeprovider

import (
	"fmt"
	"reflect"
)

func Createawsautoscalegroupv2(_params interface{}) (INodeProvider, error) {
	params, lok := _params.([]interface{})
	if !lok {
		return nil, fmt.Errorf("params is not []string")
	}

	// Преобразуем каждый аргумент в значение reflect.Value
	argValues := make([]reflect.Value, 0, len(params))
	for _, arg := range params {
		argValues = append(argValues, reflect.ValueOf(arg))
	}

	result := reflect.ValueOf(NewAwsAutoscaleGroupProvider).Call(argValues)
	if !result[1].IsNil() {
		return nil, fmt.Errorf("aws autoscale group provider returned error: %v", result[1].Interface().(error))
	}

	lprovider := result[0].Interface().(INodeProvider)
	return lprovider, nil
}
