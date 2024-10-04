package object

import (
	"fmt"
	"github.com/carmooo/monkey_compiler/code"
	"github.com/carmooo/monkey_interpreter/object"
)

const (
	COMPILED_FUNCTION_OBJECT = "COMPILED_FUNCTION_OBJECT"
)

type CompiledFunction struct {
	Instructions  code.Instructions
	NumLocals     int
	NumParameters int
}

func (cf *CompiledFunction) Type() object.ObjectType {
	return COMPILED_FUNCTION_OBJECT
}
func (cf *CompiledFunction) Inspect() string {
	return fmt.Sprintf("CompiledFunction[%p]", cf)
}
