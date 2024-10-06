package vm

import (
	"fmt"
	"github.com/carmooo/monkey_compiler/code"
	"github.com/carmooo/monkey_compiler/compiler"
	compilerObject "github.com/carmooo/monkey_compiler/object"
	"github.com/carmooo/monkey_interpreter/object"
)

const StackSize = 2048
const GlobalsSize = 65536
const MaxFrames = 1024

var True = &object.Boolean{Value: true}
var False = &object.Boolean{Value: false}

var Null = &object.Null{}

type Frame struct {
	cl *compilerObject.Closure
	ip int

	basePointer int
}

func NewFrame(cl *compilerObject.Closure, basePointer int) *Frame {
	return &Frame{cl: cl, ip: -1, basePointer: basePointer}
}

func (f *Frame) Instructions() code.Instructions {
	return f.cl.Fn.Instructions
}

type VM struct {
	constants []object.Object

	stack []object.Object
	sp    int // always points to next value. top of stakck is always stack[sp-1]

	globals []object.Object

	frames      []*Frame
	framesIndex int
}

func New(bytecode *compiler.ByteCode) *VM {
	mainFn := &compilerObject.CompiledFunction{Instructions: bytecode.Instructions}
	mainClosure := &compilerObject.Closure{Fn: mainFn}
	mainFrame := NewFrame(mainClosure, 0)

	frames := make([]*Frame, MaxFrames)
	frames[0] = mainFrame

	return &VM{
		constants: bytecode.Constants,

		stack: make([]object.Object, StackSize),
		sp:    0,

		globals: make([]object.Object, GlobalsSize),

		frames:      frames,
		framesIndex: 1,
	}
}

func NewWithGlobalsStore(bytecode *compiler.ByteCode, store []object.Object) *VM {
	vm := New(bytecode)
	vm.globals = store
	return vm
}

func (vm *VM) Run() error {
	var ip int
	var instructions code.Instructions
	var op code.Opcode

	for vm.currentFrame().ip < len(vm.currentFrame().Instructions())-1 {
		vm.currentFrame().ip++

		ip = vm.currentFrame().ip
		instructions = vm.currentFrame().Instructions()
		op = code.Opcode(instructions[ip])

		switch op {
		case code.OpConstant:
			constIndex := code.ReadUint16(instructions[ip+1:])
			vm.currentFrame().ip += 2

			err := vm.push(vm.constants[constIndex])
			if err != nil {
				return err
			}

		case code.OpAdd, code.OpSub, code.OpMul, code.OpDiv:
			err := vm.executeBinaryOperation(op)
			if err != nil {
				return err
			}

		case code.OpPop:
			vm.pop()

		case code.OpTrue:
			err := vm.push(True)
			if err != nil {
				return err
			}

		case code.OpFalse:
			err := vm.push(False)
			if err != nil {
				return err
			}

		case code.OpEqual, code.OpNotEqual, code.OpGreaterThan:
			err := vm.executeComparisonOperation(op)
			if err != nil {
				return err
			}

		case code.OpMinus:
			err := vm.executeMinusOperation()
			if err != nil {
				return err
			}

		case code.OpBang:
			err := vm.executeBangOperation()
			if err != nil {
				return err
			}

		case code.OpJump:
			pos := int(code.ReadUint16(instructions[ip+1:]))
			vm.currentFrame().ip = pos - 1

		case code.OpJumpNotTruthy:
			pos := int(code.ReadUint16(instructions[ip+1:]))
			vm.currentFrame().ip += 2

			condition := vm.pop()
			if !isTruthy(condition) {
				vm.currentFrame().ip = pos - 1
			}

		case code.OpNull:
			err := vm.push(&object.Null{})
			if err != nil {
				return err
			}

		case code.OpSetGlobal:
			globalIndex := code.ReadUint16(instructions[ip+1:])
			vm.currentFrame().ip += 2

			vm.globals[globalIndex] = vm.pop()

		case code.OpGetGlobal:
			globalIndex := code.ReadUint16(instructions[ip+1:])
			vm.currentFrame().ip += 2

			err := vm.push(vm.globals[globalIndex])
			if err != nil {
				return err
			}

		case code.OpArray:
			lenArray := int(code.ReadUint16(instructions[ip+1:]))
			vm.currentFrame().ip += 2

			array := vm.buildArray(vm.sp-lenArray, vm.sp)
			vm.sp = vm.sp - lenArray

			err := vm.push(array)
			if err != nil {
				return err
			}

		case code.OpHash:
			lenHash := int(code.ReadUint16(instructions[ip+1:]))
			vm.currentFrame().ip += 2

			hash, err := vm.buildHash(vm.sp-lenHash, vm.sp)
			if err != nil {
				return err
			}

			vm.sp = vm.sp - lenHash

			err = vm.push(hash)
			if err != nil {
				return err
			}

		case code.OpIndex:
			indexObject := vm.pop()
			leftObject := vm.pop()

			err := vm.executeIndexExpression(leftObject, indexObject)
			if err != nil {
				return err
			}

		case code.OpCall:
			numArgs := code.ReadUint8(instructions[ip+1:])
			vm.currentFrame().ip++

			err := vm.executeCall(int(numArgs))
			if err != nil {
				return err
			}

		case code.OpReturnValue:
			returnValue := vm.pop()

			frame := vm.popFrame()
			// the -1 avoids having to pop the just executed func
			vm.sp = frame.basePointer - 1

			err := vm.push(returnValue)
			if err != nil {
				return nil
			}

		case code.OpReturn:
			frame := vm.popFrame()
			// the -1 avoids having to pop the just executed func
			vm.sp = frame.basePointer - 1

			err := vm.push(Null)
			if err != nil {
				return err
			}

		case code.OpSetLocal:
			localIndex := code.ReadUint8(instructions[ip+1:])
			vm.currentFrame().ip += 1

			frame := vm.currentFrame()
			vm.stack[frame.basePointer+int(localIndex)] = vm.pop()

		case code.OpGetLocal:
			localIndex := code.ReadUint8(instructions[ip+1:])
			vm.currentFrame().ip++

			frame := vm.currentFrame()
			err := vm.push(vm.stack[frame.basePointer+int(localIndex)])
			if err != nil {
				return err
			}

		case code.OpGetBuiltin:
			builtinIndex := code.ReadUint8(instructions[ip+1:])
			vm.currentFrame().ip++

			definition := object.Builtins[builtinIndex]

			err := vm.push(definition.Builtin)
			if err != nil {
				return err
			}

		case code.OpClosure:
			constIndex := int(code.ReadUint16(instructions[ip+1:]))
			_ = code.ReadUint8(instructions[ip+1:])
			vm.currentFrame().ip += 3

			constant := vm.constants[constIndex]
			function, ok := constant.(*compilerObject.CompiledFunction)
			if !ok {
				return fmt.Errorf("not a function: %+v", constant)
			}

			closure := &compilerObject.Closure{Fn: function}
			err := vm.push(closure)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (vm *VM) currentFrame() *Frame {
	return vm.frames[vm.framesIndex-1]
}

func (vm *VM) pushFrame(f *Frame) {
	vm.frames[vm.framesIndex] = f
	vm.framesIndex++
}

func (vm *VM) popFrame() *Frame {
	vm.framesIndex--
	return vm.frames[vm.framesIndex]
}

func (vm *VM) StackTop() object.Object {
	if vm.sp == 0 {
		return nil
	}
	return vm.stack[vm.sp-1]
}

func (vm *VM) LastPoppedStackElem() object.Object {
	return vm.stack[vm.sp]
}

func (vm *VM) push(o object.Object) error {
	if vm.sp >= StackSize {
		return fmt.Errorf("stack overflow")
	}

	vm.stack[vm.sp] = o
	vm.sp++

	return nil
}

func (vm *VM) pop() object.Object {
	o := vm.stack[vm.sp-1]
	vm.sp--
	return o
}

func (vm *VM) executeBinaryOperation(op code.Opcode) error {
	right := vm.pop()
	left := vm.pop()

	leftType := left.Type()
	rightType := right.Type()

	switch {
	case leftType == object.INTEGER_OBJECT && rightType == object.INTEGER_OBJECT:
		return vm.executeBinaryIntegerOperation(op, left, right)

	case leftType == object.STRING_OBJECT && rightType == object.STRING_OBJECT:
		return vm.executeBinaryStringOperation(op, left, right)

	default:
		return fmt.Errorf("unsupported types for binary operation: %s %s",
			leftType, rightType)
	}
}

func (vm *VM) executeBinaryIntegerOperation(op code.Opcode, left, right object.Object) error {
	leftValue := left.(*object.Integer).Value
	rightValue := right.(*object.Integer).Value

	var result int64

	switch op {
	case code.OpAdd:
		result = leftValue + rightValue
	case code.OpSub:
		result = leftValue - rightValue
	case code.OpMul:
		result = leftValue * rightValue
	case code.OpDiv:
		result = leftValue / rightValue
	default:
		return fmt.Errorf("unknown integer operator: %d", op)
	}

	return vm.push(&object.Integer{Value: result})
}

func (vm *VM) executeBinaryStringOperation(op code.Opcode, left, right object.Object) error {
	leftValue := left.(*object.String).Value
	rightValue := right.(*object.String).Value

	var result string

	switch op {
	case code.OpAdd:
		result = fmt.Sprintf("%s%s", leftValue, rightValue)
	default:
		return fmt.Errorf("unknown integer operator: %d", op)
	}

	return vm.push(&object.String{Value: result})
}

func (vm *VM) executeComparisonOperation(op code.Opcode) error {
	right := vm.pop()
	left := vm.pop()

	leftType := left.Type()
	rightType := right.Type()

	if leftType == object.INTEGER_OBJECT && rightType == object.INTEGER_OBJECT {
		return vm.executeIntegerComparison(op, left, right)
	}

	switch op {
	case code.OpEqual:
		return vm.push(nativeBoolToBoolean(left == right))
	case code.OpNotEqual:
		return vm.push(nativeBoolToBoolean(left != right))
	default:
		return fmt.Errorf("unknown operator: %d", op)
	}
}

func (vm *VM) executeIntegerComparison(op code.Opcode, left, right object.Object) error {
	leftValue := left.(*object.Integer).Value
	rightValue := right.(*object.Integer).Value

	switch op {
	case code.OpEqual:
		return vm.push(nativeBoolToBoolean(leftValue == rightValue))
	case code.OpNotEqual:
		return vm.push(nativeBoolToBoolean(leftValue != rightValue))
	case code.OpGreaterThan:
		return vm.push(nativeBoolToBoolean(leftValue > rightValue))
	default:
		return fmt.Errorf("unknown operator: %d", op)
	}
}

func (vm *VM) executeMinusOperation() error {
	right := vm.pop()
	if right.Type() != object.INTEGER_OBJECT {
		return fmt.Errorf("unsopported type for negation: %s", right.Type())
	}
	rightValue := right.(*object.Integer).Value
	return vm.push(&object.Integer{Value: -rightValue})
}

func (vm *VM) executeBangOperation() error {
	right := vm.pop()
	switch right {
	case False, Null:
		return vm.push(True)
	default:
		return vm.push(False)
	}
}

func (vm *VM) buildArray(startIndex, endIndex int) object.Object {
	elements := make([]object.Object, endIndex-startIndex)

	for i := startIndex; i < endIndex; i++ {
		elements[i-startIndex] = vm.stack[i]
	}

	return &object.Array{Elements: elements}
}

func (vm *VM) buildHash(startIndex, endIndex int) (object.Object, error) {
	pairs := make(map[object.HashKey]object.HashPair)

	for i := startIndex; i < endIndex-1; i += 2 {
		key := vm.stack[i]
		value := vm.stack[i+1]

		pair := object.HashPair{
			Key:   key,
			Value: value,
		}

		hashKey, ok := key.(object.Hashable)
		if !ok {
			return nil, fmt.Errorf("unusable hash key: %s", key.Type())
		}

		pairs[hashKey.HashKey()] = pair
	}

	return &object.Hash{Pairs: pairs}, nil
}

func (vm *VM) executeIndexExpression(left, index object.Object) error {
	switch {
	case left.Type() == object.ARRAY_OBJECT && index.Type() == object.INTEGER_OBJECT:
		return vm.executeArrayIndex(left, index)
	case left.Type() == object.HASH_OBJECT:
		return vm.executeHashIndex(left, index)
	default:
		return fmt.Errorf("index operator not supported: %s", left.Type())
	}
}

func (vm *VM) executeArrayIndex(array, index object.Object) error {
	arrayObject := array.(*object.Array)
	i := index.(*object.Integer).Value
	max := int64(len(arrayObject.Elements) - 1)

	if 0 <= i && i <= max {
		return vm.push(arrayObject.Elements[i])
	}
	return vm.push(Null)
}

func (vm *VM) executeHashIndex(hash, index object.Object) error {
	hashObject := hash.(*object.Hash)

	key, ok := index.(object.Hashable)
	if !ok {
		return fmt.Errorf("unusable as hash key: %s", index.Type())
	}

	pair, ok := hashObject.Pairs[key.HashKey()]
	if !ok {
		return vm.push(Null)
	}
	return vm.push(pair.Value)
}

func (vm *VM) executeCall(numArgs int) error {
	calee := vm.stack[vm.sp-1-numArgs]
	switch calee := calee.(type) {

	case *compilerObject.Closure:
		if numArgs != calee.Fn.NumParameters {
			return fmt.Errorf("wrong number of arguments: want=%d, got=%d",
				calee.Fn.NumParameters, numArgs)
		}

		frame := NewFrame(calee, vm.sp-numArgs)
		vm.pushFrame(frame)
		// vm.sp += fn.numLocals
		vm.sp = frame.basePointer + calee.Fn.NumLocals

		return nil

	case *object.Builtin:
		args := vm.stack[vm.sp-numArgs : vm.sp]

		result := calee.Fn(args...)
		vm.sp -= numArgs + 1

		if result != nil {
			return vm.push(result)
		} else {
			return vm.push(Null)
		}

	default:
		return fmt.Errorf("calling non-function")
	}
}

func nativeBoolToBoolean(b bool) object.Object {
	if b {
		return True
	}
	return False
}

func isTruthy(o object.Object) bool {
	switch o := o.(type) {
	case *object.Boolean:
		return o.Value
	case *object.Null:
		return false
	default:
		return true
	}
}
