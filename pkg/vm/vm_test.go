package vm

import (
	"testing"
)

func TestY(t *testing.T) {
	ifErr := fatalIfErr(t)
	var cpu CPU
	cpu.maxExecSz = 20
	cpu.maxStepsPerInp = 5
	cpu.Exec.Push(Ops["not"])
	cpu.Exec.Push(Ops["if"])
	cpu.Exec.Push(Ops["y"])
	_, err := cpu.Step()
	ifErr(err, "stepping cpu")
	exp := `not
if
y
not
if
`
	if got := cpu.ExecString(); got != exp {
		t.Errorf("expected:\n%s\ngot:\n%s", exp, got)
	}
}

func TestIf(t *testing.T) {
	not := Ops["not"]
	y := Ops["y"]
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
		cpu.maxExecSz = 20
		cpu.maxStepsPerInp = 5
		cpu.Exec.Push(not)
		cpu.Exec.Push(y)
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
		cpu.maxExecSz = 20
		cpu.maxStepsPerInp = 5
		cpu.Exec.Push(not)
		cpu.Exec.Push(y)
		cpu.Exec.Push(ifi)

		cpu.Bool.Push(true)

		_, err := cpu.Step()
		ifErr(err, "stepping cpu")
		exp := `y
`
		if got := cpu.ExecString(); got != exp {
			t.Errorf("expected:\n%s\ngot:\n%s", exp, got)
		}

	})
}


func fatalIfErr(t *testing.T) func(err error, what string) {
	return func(err error, what string) {
		t.Helper()
		if err != nil {
			t.Fatalf("expected no errors when %s, got: %s", what, err)
		}
	}
}
