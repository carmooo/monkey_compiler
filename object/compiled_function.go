package object

import (
	"fmt"
	"github.com/carmooo/monkey_compiler/code"
	"github.com/carmooo/monkey_interpreter/object"
)

const (
	COMPILED_FUNCTION_OBJECT = "COMPILED_FUNCTION_OBJECT"
)

type CompiledFunctionObject struct {
	Instructions code.Instructions
}

func (cf *CompiledFunctionObject) Type() object.ObjectType {
	return COMPILED_FUNCTION_OBJECT
}
func (cf *CompiledFunctionObject) Inspect() string {
	return fmt.Sprintf("CompiledFunction[%p]", cf)
}
