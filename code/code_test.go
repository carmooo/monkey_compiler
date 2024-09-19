package code

import "testing"

func MakeTest(t *testing.T) {
	tests := []struct {
		op       Opcode
		operands []int
		expected []byte
	}{
		{OpConstant, []int{65534}, []byte{byte(OpConstant), 255, 254}},
	}

	for _, tt := range tests {
		instruction := Make(tt.op, tt.operands...)

		if len(instruction) != len(tt.expected) {
			t.Errorf("instruction has the wrong length. want=%d, got=%d",
				len(instruction), len(tt.expected))
		}

		for i, b := range tt.expected {
			if instruction[i] != b {
				t.Errorf("wrong byte at position %d.  want=%d, got=%d",
					i, b, instruction[i])
			}
		}
	}
}
