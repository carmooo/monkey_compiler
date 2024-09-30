package compiler

import (
	"fmt"
	"github.com/carmooo/monkey_compiler/code"
	"github.com/carmooo/monkey_interpreter/ast"
	"github.com/carmooo/monkey_interpreter/lexer"
	"github.com/carmooo/monkey_interpreter/object"
	"github.com/carmooo/monkey_interpreter/parser"
	"testing"
)

type compilerTestCase struct {
	input                string
	expectedInstructions []code.Instructions
	expectedConstants    []interface{}
}

func TestIntegerArithmetic(t *testing.T) {
	tests := []compilerTestCase{
		{
			input: "1 + 2",
			expectedInstructions: []code.Instructions{
				code.Make(code.OpConstant, 0),
				code.Make(code.OpConstant, 1),
				code.Make(code.OpAdd),
				code.Make(code.OpPop),
			},
			expectedConstants: []interface{}{1, 2},
		},
		{
			input:             "1; 2",
			expectedConstants: []interface{}{1, 2},
			expectedInstructions: []code.Instructions{
				code.Make(code.OpConstant, 0),
				code.Make(code.OpPop),
				code.Make(code.OpConstant, 1),
				code.Make(code.OpPop),
			},
		},
	}

	runCompilerTests(t, tests)
}

func runCompilerTests(t *testing.T, tests []compilerTestCase) {
	t.Helper()

	for _, tt := range tests {
		program := parse(tt.input)

		compiler := New()

		err := compiler.Compile(program)
		if err != nil {
			t.Errorf("compiler error: %s", err)
		}

		byteCode := compiler.ByteCode()

		err = testInstructions(tt.expectedInstructions, byteCode.Instructions)
		if err != nil {
			t.Errorf("test instructions failed: %s", err)
		}

		err = testConstants(tt.expectedConstants, byteCode.Constants)
		if err != nil {
			t.Errorf("test constants failed: %s", err)
		}
	}
}

func parse(input string) *ast.Program {
	l := lexer.New(input)
	p := parser.New(l)

	return p.ParseProgram()
}

func testInstructions(expected []code.Instructions, actual code.Instructions) error {
	concatted := concatInstructions(expected)

	if len(actual) != len(concatted) {
		return fmt.Errorf("wrong instructions length.\nwant=%q\ngot =%q",
			concatted, actual)
	}

	for i, ins := range concatted {
		if actual[i] != ins {
			return fmt.Errorf("wrong instruction at %d.\nwant=%q\ngot =%q",
				i, concatted, actual)
		}
	}

	return nil
}

func concatInstructions(instructions []code.Instructions) code.Instructions {
	out := code.Instructions{}
	for _, ins := range instructions {
		out = append(out, ins...)
	}

	return out
}

func testConstants(expected []interface{}, actual []object.Object) error {
	if len(actual) != len(expected) {
		return fmt.Errorf("wrong constants length. want=%d, got=%d",
			len(expected), len(actual))
	}

	for i, constant := range expected {
		switch constant := constant.(type) {
		case int:
			err := testIntegerObject(int64(constant), actual[i])
			if err != nil {
				return fmt.Errorf("wrong constant at %d. \nwant=%q,\n got=%q",
					i, constant, actual[i])
			}
		}
	}

	return nil
}

func testIntegerObject(expected int64, actual object.Object) error {
	result, ok := actual.(*object.Integer)
	if !ok {
		return fmt.Errorf("object is not integer. got+%T (%+v)",
			actual, actual)
	}

	if result.Value != expected {
		return fmt.Errorf("object has wrong value. want=%d, got=%d",
			expected, result.Value)
	}

	return nil
}
