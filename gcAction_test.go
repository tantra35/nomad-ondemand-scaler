package main

import (
	"reflect"
	"testing"

	"github.com/Pramod-Devireddy/go-exprtk"
)

func TestExprtkGCMaxFreeExpression(t *testing.T) {
	exprtkObj := exprtk.NewExprtk()
	exprtkObj.AddDoubleVariable("totalnodes")

	t.Logf("result: %v", reflect.TypeOf(&exprtkObj))

	exprtkObj.SetExpression("min(round(totalnodes * 0.1), 2)")

	lerr := exprtkObj.CompileExpression()
	if lerr != nil {
		t.Fatalf("can't compile expression: %s", lerr)
	}
	exprtkObj.SetDoubleVariableValue("totalnodes", float64(0))
	t.Logf("result: %d", int(exprtkObj.GetEvaluatedValue())) // exprtkObj.GetEvaluatedValue()
}

func TestExprtkIfExpression(t *testing.T) {
	exprtkObj := exprtk.NewExprtk()
	exprtkObj.AddDoubleVariable("totalnodes")
	exprtkObj.AddDoubleVariable("busynodes")

	t.Logf("result: %v", reflect.TypeOf(&exprtkObj))

	exprtkObj.SetExpression(`
	if (totalnodes < 3)
	{
	  0
	} 
	else
	{
		min(ceil(totalnodes * 0.3), 5)
	}`)

	lerr := exprtkObj.CompileExpression()
	if lerr != nil {
		t.Fatalf("can't compile expression: %s", lerr)
	}

	exprtkObj.SetDoubleVariableValue("totalnodes", float64(2))
	result := int(exprtkObj.GetEvaluatedValue())
	if result != 0 {
		t.Fatalf("wrong result: %d", result)
	}
	t.Logf("result: %d", result)

	exprtkObj.SetDoubleVariableValue("totalnodes", float64(4))
	result = int(exprtkObj.GetEvaluatedValue())
	if result != 2 {
		t.Fatalf("wrong result: %d", result)
	}
	t.Logf("result: %d", result)

	exprtkObj.SetDoubleVariableValue("totalnodes", float64(20))
	result = int(exprtkObj.GetEvaluatedValue())
	if result != 5 {
		t.Fatalf("wrong result: %d", result)
	}
	t.Logf("result: %d", result)
}
