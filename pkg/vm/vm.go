package vm

import (
	"strconv"
	"strings"

	"github.com/ORBAT/bort/pkg/fucking"
)

// general VM settings
var (
	// MaxStepsPerInput governs how many steps per each input item each individual can run. For
	// example, for an input of length 5 and MaxStepsPerInput of 4, each individual would have a
	// total of 20 steps to do its thing.
	MaxStepsPerInput = 10.0

	MaxExecStackSize = 30
)

type StackType uint8

func (st StackType) String() string {
	switch st {
	case StackExec:
		return "StackExec"
	case StackInt:
		return "StackInt"
	case StackBool:
		return "StackBool"
	case StackStack:
		return "StackStack"
	}
	panic("fuu")
}

func (st StackType) ToOpFn() OpFn {
	return func(cpu *CPU) error {
		cpu.Stack.Push(st)
		return nil
	}
}

// stack types
const (
	StackExec StackType = iota
	StackInt
	StackBool
	StackStack
)

type StackError string

func (s StackError) Error() string {
	return string(s)
}

const (
	ErrStackEmpty = StackError("stack empty")
)

type Stack []interface{}

func (s *Stack) Empty() bool {
	return len(*s) == 0
}

func (s *Stack) Len() int {
	if s == nil {
		return 0
	}
	return len(*s)
}

func (s *Stack) Peek() (interface{}, error) {
	slen := len(*s)
	if slen == 0 {
		return nil, ErrStackEmpty
	}
	return (*s)[slen-1], nil
}

func (s *Stack) Push(i interface{}) {
	*s = append(*s, i)
}

func (s *Stack) Drop() {
	slen := len(*s)
	if slen == 0 {
		return
	}
	*s = (*s)[:slen-1]
}

// ( a b -- a b a )
func (s *Stack) Over() {
	slen := len(*s)
	if slen < 2 {
		return
	}
	second := (*s)[slen-2]
	*s = append(*s, second)
}

func (s *Stack) Nip() {
	s.Swap()
	s.Drop()
}

func (s *Stack) Tuck() {
	s.Swap()
	s.Over()
}

func (s *Stack) Pop() (interface{}, error) {
	slen := len(*s)
	if slen == 0 {
		return nil, ErrStackEmpty
	}

	var top interface{}
	top, *s = (*s)[slen-1], (*s)[:slen-1]

	return top, nil
}

func (s *Stack) Reset() {
	*s = []interface{}{}
}

// Unorthodox rot for the entire stack, not just top 3: ( a b c d -- d a b c )
func (s *Stack) Rot() {
	if len(*s) < 2 {
		return
	}
	top, _ := s.Pop()
	*s = append([]interface{}{top}, *s...)
}

// Classic Rot   ( a b c -- b c a )
func (s *Stack) Rot3() {
	if len(*s) < 3 {
		return
	}
	last3Idx := s.Len() - 3
	last3 := (*s)[last3Idx:]
	last3[0], last3[1], last3[2] = last3[1], last3[2], last3[0]
}

func (s *Stack) Dup() {
	if len(*s) == 0 {
		return
	}
	v, _ := s.Peek()
	s.Push(v)
}

func (s *Stack) Swap() {
	if len(*s) < 2 {
		return
	}
	(*s)[0], (*s)[1] = (*s)[1], (*s)[0]
}

func (s *Stack) CopyAt(i int) {
	slen := len(*s)
	if slen < 2 {
		return
	}
	if i < 0 {
		i *= -1
	}
	idx := i % slen
	*s = append(*s, (*s)[idx])
}

func (s *Stack) Shove(i int) {
	slen := len(*s)
	if slen < 2 {
		return
	}
	if i < 0 {
		i *= -1
	}

	// shove top of stack into idx
	top, _ := s.Pop()
	slen--
	idx := i % slen
	*s = append(*s, nil)
	copy((*s)[idx+1:], (*s)[idx:])
	(*s)[idx] = top
}

func (s *Stack) Yank(i int) {
	slen := len(*s)
	if slen < 2 {
		return
	}
	if i < 0 {
		i *= -1
	}
	idx := i % slen
	it := (*s)[idx]
	*s = append(append((*s)[:idx], (*s)[idx+1:]...), it)
}

type Stacks struct {
	// Exec should contain Ops
	Exec Stack

	Int  Stack
	Bool Stack

	// Stack should contain StackTypes
	Stack Stack
}

func (s *Stacks) Reset() {
	s.Exec.Reset()
	s.Int.Reset()
	s.Bool.Reset()
	s.Stack.Reset()
}

// PopStack pops from s.Stack, or returns StackInt if s.Stack is empty
func (s *Stacks) PopStack() StackType {
	top, err := s.Stack.Pop()
	if err != nil {
		top = StackInt
	}
	return top.(StackType)
}

// PopExec pops from s.Exec. Panics if Exec is empty
func (s *Stacks) PopExec() Op {
	iop, err := s.Exec.Pop()
	if err != nil {
		panic(err)
	}
	return iop.(Op)
}

// PeekExec peeks s.Exec. Panics if Exec is empty
func (s *Stacks) PeekExec() Op {
	return s.Exec[len(s.Exec)-1].(Op)
}

func (s *Stacks) OfType(t StackType) *Stack {
	switch t {
	case StackExec:
		return &s.Exec
	case StackInt:
		return &s.Int
	case StackBool:
		return &s.Bool
	case StackStack:
		return &s.Stack
	}
	panic("wtf kind of StackType is " + strconv.Itoa(int(t)))
}

type CPUError string

func (ce CPUError) Error() string {
	return string(ce)
}

const (
	ErrStepsExceeded = CPUError("Too many steps taken")
	ErrExecOverflow  = CPUError("Exec stack too large")
)

func NewCPU(exec []Op) *CPU {
	return &CPU{
		Stacks: Stacks{
			Exec: fucking.InterfaceSlice(exec),
		},
		rom: fucking.InterfaceSlice(exec),
	}
}

type Op struct {
	Name string
	fn   OpFn
}

type OpFn func(*CPU) error

func (fn OpFn) Op(n string) Op {
	return Op{
		n,
		fn,
	}
}

// CPU that executes Ops. The zero value is usable.
type CPU struct {
	Stacks
	NSteps uint32
	input  []int
	rom    []interface{}
	inpLen int
	halt   bool

	isp1, isp2 uint16
}

func (c *CPU) IncrISP1() error {
	c.isp1++
	return nil
}

func (c *CPU) IncrISP2() error {
	c.isp2++
	return nil
}

func (c *CPU) DecrISP1() error {
	c.isp1--
	return nil
}

func (c *CPU) DecrISP2() error {
	c.isp2--
	return nil
}

func (c *CPU) ispToIdx(isp uint16) int {
	return int(isp) % (c.Int.Len() - 1)
}

func (c *CPU) SwapISPs() error {
	if c.Int.Len() < 2 {
		return CPUError("Can't swap ISPs when int stack len < 2")
	}
	a, b := c.ispToIdx(c.isp1), c.ispToIdx(c.isp2)
	c.Int[a], c.Int[b] = c.Int[b], c.Int[a]
	return nil
}

//   bool: ( -- c.Int[isp1] < c.Int[isp2] )
func (c *CPU) LTISPs() error {
	if c.Int.Len() < 2 {
		return CPUError("Can't swap ISPs when int stack len < 2")
	}
	a, b := c.ispToIdx(c.isp1), c.ispToIdx(c.isp2)
	c.Bool.Push(c.Int[a].(int) < c.Int[b].(int))
	return nil
}

func (c CPU) Clone() *CPU {
	return &c
}

func (c *CPU) Input(input []int) *CPU {
	c.input = input
	c.inpLen = len(input)
	return c.Reset()
}

func (c *CPU) OrigInput() []int {
	return c.input
}

func (c *CPU) Reset() *CPU {
	c.Stacks.Reset()
	c.Exec = make([]interface{}, len(c.rom))
	copy(c.Exec, c.rom)
	c.Int = fucking.InterfaceSlice(c.input)
	c.NSteps = 0
	return c
}

func (c *CPU) ExecString() string {
	if c.Exec.Len() == 0 {
		return "<no code in exec>"
	}
	var b strings.Builder
	for _, opiface := range c.Exec {
		op := opiface.(Op)
		b.WriteString(op.Name)
		b.WriteRune('\n')
	}
	return b.String()
}

func (c *CPU) shouldStep() error {
	if c.NSteps >= uint32(float64(c.inpLen)*MaxStepsPerInput) {
		return ErrStepsExceeded
	}

	if c.Exec.Len() > MaxExecStackSize {
		return ErrExecOverflow
	}
	return nil
}

// Step runs one step of the CPU. Returns true,nil if the Exec stack is empty or halt is executed. err has the error (if
// any) returned by the last op
func (c *CPU) Step() (execDone bool, err error) {
	if len(c.Exec) == 0 || c.halt {
		return true, nil
	}

	op := c.PeekExec()
	err = op.fn(c)
	c.Exec.Drop()
	c.NSteps++
	return false, err
}

func (c *CPU) Run(ignoreStepErrs bool) (completed bool, err error) {
	c.Reset()
	for {
		if err := c.shouldStep(); err != nil {
			if ignoreStepErrs == true {
				err = nil
			}
			return false, err
		}
		done, err := c.Step()

		if done {
			return true, nil
		}
		if err != nil && !ignoreStepErrs {
			return false, err
		}
	}
}

type rawOpMap map[string]OpFn

func (pm rawOpMap) ToOps() (ops map[string]Op) {
	ops = make(map[string]Op, len(pm))
	for name, fn := range pm {
		ops[name] = Op{name, fn}
	}
	return
}

// Y combinator(ish) operator
//   ( a b c d y -- a b c d y a b c d )
//
// y basically copies the Exec stack, and prepends the copy (along with itself) back to Exec
func y(cpu *CPU) error {
	// if exec only has y on it we might as well nuke it and call it a day since that'd be an
	// infinite loop
	if len(cpu.Exec) == 1 {
		cpu.Exec.Reset()
		return nil
	}

	clone := make(Stack, len(cpu.Exec))
	// include the topmost y in the clone, since the top of exec gets popped after each step anyhow
	copy(clone, cpu.Exec)
	cpu.Exec = append(cpu.Exec, clone...)
	if cpu.Exec.Len() > MaxExecStackSize {
		cpu.Exec = cpu.Exec[:MaxExecStackSize]
	}
	return nil
}

type StackFn func(*Stack)

func (fn StackFn) ToOpFn() OpFn {
	return func(cpu *CPU) error {
		fn(cpu.OfType(cpu.PopStack()))
		return nil
	}
}

var Ops = rawOpMap{
	"len": func(cpu *CPU) error {
		stack := cpu.PopStack()
		cpu.Int.Push(cpu.OfType(stack).Len())
		return nil
	},
	// "rot": StackFn((*Stack).Rot).ToOpFn(),
	// "rot3":  StackFn((*Stack).Rot3).ToOpFn(),
	"dup":  StackFn((*Stack).Dup).ToOpFn(),
	"swap": StackFn((*Stack).Swap).ToOpFn(),
	"over": StackFn((*Stack).Over).ToOpFn(),
	"nip":  StackFn((*Stack).Over).ToOpFn(),
	"tuck": StackFn((*Stack).Over).ToOpFn(),
	"reset": StackFn((*Stack).Reset).ToOpFn(),
	"drop": StackFn((*Stack).Drop).ToOpFn(),
	// "yank": func(cpu *CPU) error {
	// 	stack := cpu.PopStack()
	// 	stackToYank := cpu.OfType(stack)
	// 	if stackToYank.Len() == 0 {
	// 		return CPUError("yank of empty stack " + stack.String())
	// 	}
	// 	nToYank, err := cpu.Int.Pop()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	stackToYank.Yank(nToYank.(int))
	// 	return nil
	// },
	//
	// "shove": func(cpu *CPU) error {
	// 	stack := cpu.PopStack()
	// 	stackToShove := cpu.OfType(stack)
	// 	if stackToShove.Len() == 0 {
	// 		return CPUError("shove of empty stack " + stack.String())
	// 	}
	// 	nToShove, err := cpu.Int.Pop()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	stackToShove.Shove(nToShove.(int))
	// 	return nil
	// },
	//
	// "copyat": func(cpu *CPU) error {
	// 	stack := cpu.PopStack()
	// 	stackToCopy := cpu.OfType(stack)
	// 	if stackToCopy.Len() == 0 {
	// 		return CPUError("copyat of empty stack " + stack.String())
	// 	}
	// 	nToCopy, err := cpu.Int.Pop()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	stackToCopy.CopyAt(nToCopy.(int))
	// 	return nil
	// },
	//
	//
	// "add": func(cpu *CPU) error {
	// 	// int: (a b -- a+b)
	// 	if cpu.Int.Len() < 2 {
	// 		return CPUError("Int stack len wasn't 2")
	// 	}
	// 	b, _ := cpu.Int.Pop()
	// 	a, _ := cpu.Int.Pop()
	// 	cpu.Int.Push(a.(int) + b.(int))
	// 	return nil
	// },
	//
	// "mul": func(cpu *CPU) error {
	// 	// int: (a b -- a*b)
	// 	if cpu.Int.Len() < 2 {
	// 		return CPUError("Int stack len wasn't 2")
	// 	}
	// 	b, _ := cpu.Int.Pop()
	// 	a, _ := cpu.Int.Pop()
	// 	cpu.Int.Push(a.(int) * b.(int))
	// 	return nil
	// },

	// "true": func(cpu *CPU) error {
	// 	cpu.Bool.Push(true)
	// 	return nil
	// },
	//
	// "false": func(cpu *CPU) error {
	// 	cpu.Bool.Push(false)
	// 	return nil
	// },

	// "exec":  StackExec.ToOpFn(),
	"bool":  StackBool.ToOpFn(),
	"stack": StackBool.ToOpFn(),
	"int":   StackInt.ToOpFn(),
	//
	// "lt": func(cpu *CPU) error {
	// 	// int: ( a b -- )
	// 	// bool: ( -- a > b )
	// 	if cpu.Int.Len() < 2 {
	// 		return CPUError("Int stack len wasn't 2")
	// 	}
	// 	b, _ := cpu.Int.Pop()
	// 	a, _ := cpu.Int.Pop()
	// 	cpu.Bool.Push(a.(int) < b.(int))
	// 	return nil
	// },
	//
	// "gt": func(cpu *CPU) error {
	// 	// int: ( a b -- )
	// 	// bool: ( -- a > b )
	// 	if cpu.Int.Len() < 2 {
	// 		return CPUError("Int stack len wasn't 2")
	// 	}
	// 	b, _ := cpu.Int.Pop()
	// 	a, _ := cpu.Int.Pop()
	// 	cpu.Bool.Push(a.(int) > b.(int))
	// 	return nil
	// },

	// bool: (a b -- a && b)
	"and": func(cpu *CPU) error {
		b, err := cpu.Bool.Pop()
		if err != nil {
			return err
		}
		a, err := cpu.Bool.Pop()
		if err != nil {
			return err
		}
		cpu.Bool.Push(a.(bool) && b.(bool))
		return nil
	},

	// bool: (a b -- a || b)
	"or": func(cpu *CPU) error {
		b, err := cpu.Bool.Pop()
		if err != nil {
			return err
		}
		a, err := cpu.Bool.Pop()
		if err != nil {
			return err
		}
		cpu.Bool.Push(a.(bool) || b.(bool))
		return nil
	},

	"not": func(cpu *CPU) error {
		b, err := cpu.Bool.Pop()
		if err != nil {
			return err
		}
		cpu.Bool.Push(!(b.(bool)))
		return nil
	},

	// the syntax is
	// exec: ( elseCmd thenCmd if -- )
	// bool: ( a -- )
	//
	// exec: ( cmd1 cmd2 if -- cmd1 )
	// bool: ( false -- )
	//
	// exec: ( cmd1 cmd2 if -- cmd2 )
	// bool: ( true --)
	"if": func(cpu *CPU) error {
		if cpu.Exec.Len() < 3 {
			return CPUError("Exec stack len wasn't at least 3")
		}

		b, err := cpu.Bool.Pop()
		if err != nil {
			return err
		}

		i := cpu.Exec.Len() - 2
		if b.(bool) == true {
			i = cpu.Exec.Len() - 3
		}

		cpu.Exec = append(cpu.Exec[:i], cpu.Exec[i+1:]...)
		return nil
	},

	"y": y,
	//
	// "repeat": func(cpu *CPU) error {
	// 	if cpu.Exec.Len() < 2 {
	// 		return CPUError("Exec stack len wasn't at least 2")
	// 	}
	// 	iiface, err := cpu.Int.Pop()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	count := iiface.(int)
	// 	if count+cpu.Exec.Len() > MaxExecStackSize {
	// 		return CPUError("Exec stack would overflow from this repeat")
	// 	}
	// 	toRepeat := cpu.Exec[cpu.Exec.Len()-2]
	// 	for i := 0; i < count; i++ {
	// 		cpu.Exec.Push(toRepeat)
	// 	}
	// 	return nil
	// },

	// "halt": func(cpu *CPU) error {
	// 	cpu.halt = true
	// 	return nil
	// },

	"halt_sorted": func(cpu *CPU) error {
		if len(cpu.Int) == 0 {
			return nil
		}
		prevInt := cpu.Int[0].(int)
		for _, iint := range cpu.Int {
			if i := iint.(int); i < prevInt {
				return nil
			} else {
				prevInt = i
			}
		}

		cpu.halt = true
		return nil
	},

	"incr_isp1": (*CPU).IncrISP1,
	"incr_isp2": (*CPU).IncrISP2,
	"decr_isp1": (*CPU).DecrISP1,
	"decr_isp2": (*CPU).DecrISP2,
	"lt_isps":   (*CPU).LTISPs,
	"swap_isps": (*CPU).SwapISPs,
}.ToOps()

// func init() {
// 	for i := 0; i < 100; i++ {
// 		n := strconv.Itoa(i)
// 		Ops[n] = Op{
// 			Name: n,
// 			fn: func(cpu *CPU) error {
// 				cpu.Int.Push(i)
// 				return nil
// 			},
// 		}
// 	}
// }
