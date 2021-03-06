package vm

import (
	"strconv"
	"strings"

	"github.com/ORBAT/bort/pkg/config"
	"github.com/ORBAT/bort/pkg/fucking"
)

type ifaceSlice = []interface{}

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

type Stack ifaceSlice

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

func (s *Stack) Replace(is []interface{}) {
	*s = is
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
	*s = (*s)[:0] // islice{}
}

// Unorthodox rot for the entire stack, not just top 3: ( a b c d -- d a b c )
func (s *Stack) Rot() {
	if len(*s) < 2 {
		return
	}
	top, _ := s.Pop()
	*s = append(ifaceSlice{top}, *s...)
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
	idx := i % (slen - 1)
	*s = append(*s, (*s)[idx])
}

func (s *Stack) Shove(i int) {
	slen := len(*s)
	if slen < 3 {
		return
	}
	if i < 0 {
		i *= -1
	}

	// shove top of stack into idx
	top, _ := s.Pop()
	slen--
	idx := i % (slen - 1)
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
	idx := i % (slen - 1)
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

	// Cplx should contain complex128s
	Cplx Stack
}

func (s *Stacks) Reset() {
	s.Exec.Reset()
	s.Int.Reset()
	s.Bool.Reset()
	s.Stack.Reset()
}

// PopStack pops from s.Stack, or returns StackBool if s.Stack is empty
func (s *Stacks) PopStack() StackType {
	top, err := s.Stack.Pop()
	if err != nil {
		top = StackBool
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

func NewCPU(exec []Op, cfg config.CPU) *CPU {
	return &CPU{
		Stacks: Stacks{
			Exec: fucking.InterfaceSlice(exec),
		},
		rom:            fucking.InterfaceSlice(exec),
		maxExecSz:      cfg.MaxExecStackSize,
		maxStepsPerInp: cfg.MaxStepsPerInput,
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
	// TODO: make input & output stack configurable
	Stacks
	Stats
	Err            error
	maxExecSz      int
	maxStepsPerInp float64

	input ifaceSlice
	rom   ifaceSlice
	halt  bool

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
	return int(isp) % c.Int.Len()
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
		c.Bool.Push(false)
		return nil
	}
	a, b := c.ispToIdx(c.isp1), c.ispToIdx(c.isp2)
	c.Bool.Push(c.Int[a].(int) < c.Int[b].(int))
	return nil
}

func (c CPU) Clone() *CPU {
	return &c
}

func copyOf(is ifaceSlice) ifaceSlice {
	return append(make(ifaceSlice, 0,len(is)), is...)
}

func (c *CPU) Input(input []int, stack StackType) *CPU {
	c.Reset()
	c.input = fucking.InterfaceSlice(input)
	c.OfType(stack).Replace(copyOf(c.input))
	c.InpLen = len(c.input)
	return c
}

func (c *CPU) OrigInput() []interface{} {
	return c.input
}

func (c *CPU) Reset() *CPU {
	c.Stacks.Reset()
	c.Stats.Reset()
	c.Err = nil
	c.resetState()
	return c
}

type Stats struct {
	NSteps, InpLen int
}

func (s *Stats) Reset() {
	s.NSteps = 0
	s.InpLen = 0
}

func (c *CPU) resetState() {
	c.Exec = copyOf(c.rom)
	c.NSteps = 0
	c.halt = false
	c.isp1 = 0
	c.isp2 = 0
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
	if c.NSteps >= int(float64(c.InpLen)*c.maxStepsPerInp) {
		return ErrStepsExceeded
	}

	if c.Exec.Len() > c.maxExecSz {
		return ErrExecOverflow
	}
	return nil
}

// Step runs one step of the CPU. Returns true,nil if halt is executed. err has the error (if
// any) returned by the last op
func (c *CPU) Step() (execDone bool, err error) {
	// if len(c.Exec) == 0 {
	// 	c.resetState()
	// } else if c.halt {
	// 	return true, nil
	// }
	if len(c.Exec) == 0 || c.halt {
		return true, nil
	}
	op := c.PeekExec()
	err = op.fn(c)
	c.Exec.Drop()
	c.NSteps++
	return c.halt, err
}

func (c *CPU) Run(fatalErrs bool) (completed bool, err error) {
	defer func() {
		if err != nil {
			c.Err = err
		}
	}()
	for {
		if err := c.shouldStep(); err != nil {
			if !fatalErrs {
				err = nil
			}
			return false, err
		}
		done, err := c.Step()

		if done {
			return true, nil
		}
		if err != nil && fatalErrs {
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

// Y combinator(ish) operator. Does this to the exec stack:
//   ( a b y c d y -- a b y c d y c d )
//
// y copies everything between two y operators (or the the y and the bottom of the stack) and
// appends it to exec
func y(cpu *CPU) error {
	// if exec only has y on it we might as well nuke it and call it a day since that'd be an
	// infinite loop
	nExec := len(cpu.Exec)
	if nExec == 1 {
		cpu.Exec.Reset()
		return nil
	}
	nextY := nExec - 2
	for ; nextY >= 0; nextY-- {
		if cpu.Exec[nextY].(Op).Name == "y" {
			// if we find another y in exec, we want to start copying immediately _after_ it;
			// basically the ( c d y -- ) bit in the stack diagram in the doc comment
			nextY += 1
			break
		}
	}

	if nextY < 0 {
		nextY = 0
	}

	clone := make(Stack, (nExec)-nextY)
	// include the topmost y in the clone since exec gets popped after each step
	toCopy := cpu.Exec[nextY : nExec]
	copy(clone, toCopy)
	cpu.Exec = append(cpu.Exec, clone...)
	if exl, maxsz := cpu.Exec.Len(), cpu.maxExecSz; exl > maxsz {
		// preserve the top of the stack when truncating
		cpu.Exec = cpu.Exec[exl-maxsz:]
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
	// "int_len": func(cpu *CPU) error {
	// 	cpu.Int.Push(cpu.Int.Len())
	// 	return nil
	// },
	// "len": func(cpu *CPU) error {
	// 	cpu.Int.Push(cpu.OfType(cpu.PopStack()).Len())
	// 	return nil
	// },
	// "rot":   StackFn((*Stack).Rot).ToOpFn(),
	"rot3":  StackFn((*Stack).Rot3).ToOpFn(),
	"dup":   StackFn((*Stack).Dup).ToOpFn(),
	"swap":  StackFn((*Stack).Swap).ToOpFn(),
	"over":  StackFn((*Stack).Over).ToOpFn(),
	"nip":   StackFn((*Stack).Over).ToOpFn(),
	"tuck":  StackFn((*Stack).Over).ToOpFn(),
	// "reset": StackFn((*Stack).Reset).ToOpFn(),
	// "drop":  StackFn((*Stack).Drop).ToOpFn(),
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
	// "1": func(cpu *CPU) error {
	// 	cpu.Int.Push(1)
	// 	return nil
	// },
	//
	// "2": func(cpu *CPU) error {
	// 	cpu.Int.Push(2)
	// 	return nil
	// },
	//
	// "10": func(cpu *CPU) error {
	// 	cpu.Int.Push(10)
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
	// "add": func(cpu *CPU) error {
	// 	// int: (a b -- a+b)
	// 	b, err := cpu.Int.Pop()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	a, err := cpu.Int.Pop()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	cpu.Int.Push(a.(int) + b.(int))
	// 	return nil
	// },
	//
	// "sub": func(cpu *CPU) error {
	// 	// int: (a b -- a-b)
	// 	b, err := cpu.Int.Pop()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	a, err := cpu.Int.Pop()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	cpu.Int.Push(a.(int) - b.(int))
	// 	return nil
	// },
	//
	// "div": func(cpu *CPU) error {
	// 	// int: (a b -- a/b)
	// 	b, err := cpu.Int.Pop()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if b.(int) == 0 {
	// 		return CPUError("division by zero")
	// 	}
	// 	a, err := cpu.Int.Pop()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	cpu.Int.Push(a.(int) / b.(int))
	// 	return nil
	// },
	//
	// "mul": func(cpu *CPU) error {
	// 	// int: (a b -- a*b)
	// 	if cpu.Int.Len() < 2 {
	// 		return CPUError("Int stack len wasn't 2")
	// 	}
	// 	b, err := cpu.Int.Pop()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	a, err := cpu.Int.Pop()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	cpu.Int.Push(a.(int) * b.(int))
	// 	return nil
	// },
	//
	// "mod": func(cpu *CPU) error {
	// 	// int: (a b -- a%b)
	// 	b, err := cpu.Int.Pop()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	if b.(int) == 0 {
	// 		return CPUError("division by zero")
	// 	}
	// 	a, err := cpu.Int.Pop()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	cpu.Int.Push(a.(int) % b.(int))
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

	"exec": StackExec.ToOpFn(),
	"bool": StackBool.ToOpFn(),
	"stack": StackBool.ToOpFn(),
	"int": StackInt.ToOpFn(),
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
	// 	// int: ( a b --)
	// 	// bool: ( -- a > b )
	// 	b, err := cpu.Int.Pop()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	a, err := cpu.Int.Pop()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	cpu.Bool.Push(a.(int) > b.(int))
	// 	return nil
	// },

	// // bool: (a b -- a && b)
	// "and": func(cpu *CPU) error {
	// 	b, err := cpu.Bool.Pop()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	a, err := cpu.Bool.Pop()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	cpu.Bool.Push(a.(bool) && b.(bool))
	// 	return nil
	// },
	//
	// // bool: (a b -- a || b)
	// "or": func(cpu *CPU) error {
	// 	b, err := cpu.Bool.Pop()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	a, err := cpu.Bool.Pop()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	cpu.Bool.Push(a.(bool) || b.(bool))
	// 	return nil
	// },

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
		execLen := cpu.Exec.Len()
		if execLen < 3 {
			return CPUError("Exec stack len wasn't at least 3")
		}

		b, err := cpu.Bool.Pop()
		if err != nil {
			b = false
		}

		i := execLen - 2
		if b.(bool) == true {
			i -= 1
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
	// 	if count+cpu.Exec.Len() > maxExecSz {
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

	"sorted": func(cpu *CPU) error {
		if len(cpu.Int) == 0 {
			return nil
		}
		prevInt := cpu.Int[0].(int)
		for _, iint := range cpu.Int {
			if i := iint.(int); i < prevInt {
				cpu.Bool.Push(false)
			} else {
				prevInt = i
			}
		}
		cpu.Bool.Push(true)
		return nil
	},

	"halt_if_sorted": func(cpu *CPU) error {
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

	"nop": func(cpu *CPU) error {
		return nil
	},

	// "sorted": func(cpu *CPU) error {
	// 	if len(cpu.Int) == 0 {
	// 		return nil
	// 	}
	// 	prevInt := cpu.Int[0].(int)
	// 	for _, iint := range cpu.Int {
	// 		if i := iint.(int); i < prevInt {
	// 			cpu.Bool.Push(false)
	// 			return nil
	// 		} else {
	// 			prevInt = i
	// 		}
	// 	}
	//
	// 	cpu.Bool.Push(true)
	// 	return nil
	// },

	"push_isp1": func(cpu *CPU) error {
		cpu.Int.Push(int(cpu.isp1))
		return nil
	},

	"push_isp2": func(cpu *CPU) error {
		cpu.Int.Push(int(cpu.isp2))
		return nil
	},

	"incr_isp1": (*CPU).IncrISP1,
	"incr_isp2": (*CPU).IncrISP2,
	"decr_isp1": (*CPU).DecrISP1,
	"decr_isp2": (*CPU).DecrISP2,
	"lt_isps":   (*CPU).LTISPs,
	"swap_isps": (*CPU).SwapISPs,
}.ToOps()
