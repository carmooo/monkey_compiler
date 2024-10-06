package object

import (
	"fmt"
	"github.com/carmooo/monkey_interpreter/object"
)

const (
	CLOSURE_OBJECT = "CLOSURE_OBJECT"
)

type Closure struct {
	Fn            *CompiledFunction
	FreeVariables []object.Object
}

func (c *Closure) Type() object.ObjectType { return CLOSURE_OBJECT }
func (c *Closure) Inspect() string {
	return fmt.Sprintf("Closure[%p]", c)
}
