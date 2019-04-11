package vm

import (
	"fmt"
	"testing"
)

func TestY(t *testing.T) {
	ifErr := fatalIfErr(t)
	var cpu CPU
	cpu.Exec.Push(Ops["not"])
	cpu.Exec.Push(Ops["rot"])
	cpu.Exec.Push(Ops["y"])
	_, err := cpu.Step()
	ifErr(err, "stepping cpu")
	exp := `not
rot
y
not
rot
`
	if got := cpu.ExecString(); got != exp {
		t.Errorf("expected:\n%s\ngot:\n%s", exp, got)
	}
}

func TestIf(t *testing.T) {
	not := Ops["not"]
	rot := Ops["rot"]
	ifi := Ops["if"]

	// the syntax is
	// exec: ( elseCmd thenCmd if --)
	//
	// exec: (cmd1 cmd2 if -- cmd1)
	// bool: (false -- )
	//
	// exec: (cmd1 cmd2 if -- cmd2)
	// bool: (true --)

	t.Run("false", func(t *testing.T) {
		ifErr := fatalIfErr(t)

		var cpu CPU
		cpu.Exec.Push(not)
		cpu.Exec.Push(rot)
		cpu.Exec.Push(ifi)

		cpu.Bool.Push(false)

		_, err := cpu.Step()
		ifErr(err, "stepping cpu")
		exp := `not
`
		if got := cpu.ExecString(); got != exp {
			t.Errorf("expected:\n%s\ngot:\n%s", exp, got)
		}
	})

	t.Run("true", func(t *testing.T) {
		ifErr := fatalIfErr(t)

		var cpu CPU
		cpu.Exec.Push(not)
		cpu.Exec.Push(rot)
		cpu.Exec.Push(ifi)

		cpu.Bool.Push(true)

		_, err := cpu.Step()
		ifErr(err, "stepping cpu")
		exp := `rot
`
		if got := cpu.ExecString(); got != exp {
			t.Errorf("expected:\n%s\ngot:\n%s", exp, got)
		}

	})
}

func TestRot3(t *testing.T) {
	ifErr := fatalIfErr(t)
	rot3 := Ops["rot3"]
	var cpu CPU
	cpu.Exec.Push(rot3)
	cpu.Int.Push(1)
	cpu.Int.Push(2)
	cpu.Int.Push(3)
	cpu.Int.Push(4)

	_, err := cpu.Step()
	ifErr(err, "stepping cpu")
	exp := fmt.Sprintf("%v", []int{1, 3, 4, 2})
	got := fmt.Sprintf("%v", cpu.Int)
	if exp != got {
		t.Errorf("expected %s, got %s", exp, got)
	}

}

func TestRot(t *testing.T) {
	ifErr := fatalIfErr(t)
	rot := Ops["rot"]
	var cpu CPU
	cpu.Exec.Push(rot)
	cpu.Int.Push(1)
	cpu.Int.Push(2)
	cpu.Int.Push(3)
	cpu.Int.Push(4)

	_, err := cpu.Step()
	ifErr(err, "stepping cpu")
	exp := fmt.Sprintf("%v", []int{4, 1, 2, 3})
	got := fmt.Sprintf("%v", cpu.Int)
	if exp != got {
		t.Errorf("expected %s, got %s", exp, got)
	}

}

func fatalIfErr(t *testing.T) func(err error, what string) {
	return func(err error, what string) {
		t.Helper()
		if err != nil {
			t.Fatalf("expected no errors when %s, got: %s", what, err)
		}
	}
}
