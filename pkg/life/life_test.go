package life

import (
	"reflect"
	"testing"

	"github.com/ORBAT/bort/pkg/config"
	"github.com/ORBAT/bort/pkg/fucking"
	"github.com/ORBAT/bort/pkg/vm"
)

var cfg = &config.Options{
	TournamentRatio: 0,
	TournamentP:     0,
	CrossoverMutP:   1,
	PointMutP:       0,
	TransposeMutP:   0,
	ErrThreshold:    0,
	CPU: config.CPU{
		MaxExecStackSize: 25,
		MaxStepsPerInput: 5,
	},
}

func TestCross(t *testing.T) {
	// cg := CritterGenerator(20, time.Now().UnixNano())
	cr1 := NewCritter([]vm.Op{vm.Ops["y"], vm.Ops["rot3"], vm.Ops["y"], vm.Ops["if"], vm.Ops["y"], vm.Ops["if"], vm.Ops["y"], vm.Ops["if"], vm.Ops["y"], vm.Ops["if"], vm.Ops["y"], vm.Ops["if"]}, cfg)
	cr2 := NewCritter([]vm.Op{vm.Ops["not"], vm.Ops["and"], vm.Ops["not"], vm.Ops["and"], vm.Ops["not"], vm.Ops["and"], vm.Ops["not"], vm.Ops["and"], vm.Ops["not"], vm.Ops["and"], vm.Ops["not"], vm.Ops["and"]}, cfg)
	offs,_ := cr1.Cross(NewRNG(1), cr2, cfg)
	t.Log("cr1", cr1.ExecString())
	t.Log("cr2", cr2.ExecString())
	t.Log("offs", offs.ExecString())
}

func TestPos(t *testing.T) {
	// orig: [99 5 6 1 4 -555 1 0]
	// 	now:  []
	// 	want: [-555 0 1 1 4 5 6 99]
	gots := []int{0,2,7,11,11,13,14,15}
	want := []int{1,2,3,4,5,6,7,9}
	t.Log(positionalError(want, gots))
}

func TestNondeter(t *testing.T) {
	// orig := []int{11, 10, 2, 0, 14}
	is := []string{
		"nop",
		"nop",
		"lt_isps",
		"nop",
		"incr_isp2",
		"swap_isps",
		"not",
		"not",
		"lt_isps",
		"incr_isp2",
		"incr_isp2",
		"incr_isp2",
		"y",
		"not",
		"swap_isps",
		"not",
		"sorted",
		"incr_isp1",
		"lt_isps",
		"decr_isp1",
		"nop",
		"swap_isps",
		"sorted",
		"sorted",
		"decr_isp2"}
	var genome []vm.Op
	for _, insName := range is {
		genome = append(genome, vm.Ops[insName])
	}
	c := NewCritter(genome, cfg)
	c.Input([]int{11, 10, 2, 0, 14}, vm.StackInt).Run(true)
	out1 := fucking.IntSlice(c.Int)
	c.Input([]int{11, 10, 2, 0, 14}, vm.StackInt).Run(true)
	out2 := fucking.IntSlice(c.Int)

	if !reflect.DeepEqual(out1, out2) {
		t.Fatalf("expected outputs to be identical, but they weren't:\n%+v\n%+v", out1, out2)
	}
}
