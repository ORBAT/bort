package life

import (
	"testing"

	"github.com/ORBAT/bort/pkg/vm"
)

func TestCross(t *testing.T) {
	// cg := CritterGenerator(20, time.Now().UnixNano())
	cr1 := NewCritter([]vm.Op{vm.Ops["rot"], vm.Ops["rot3"], vm.Ops["rot"], vm.Ops["rot3"], vm.Ops["rot"], vm.Ops["rot3"], vm.Ops["rot"], vm.Ops["rot3"], vm.Ops["rot"], vm.Ops["rot3"], vm.Ops["rot"], vm.Ops["rot3"]})
	cr2 := NewCritter([]vm.Op{vm.Ops["not"], vm.Ops["and"], vm.Ops["not"], vm.Ops["and"], vm.Ops["not"], vm.Ops["and"], vm.Ops["not"], vm.Ops["and"], vm.Ops["not"], vm.Ops["and"], vm.Ops["not"], vm.Ops["and"]})
	offs := cr1.Cross(cr2, NewRNG(1))
	t.Log("cr1", cr1.ExecString())
	t.Log("cr2", cr2.ExecString())
	t.Log("offs", offs.ExecString())
}

func TestMutate(t *testing.T) {
	cr1 := NewCritter([]vm.Op{vm.Ops["rot"], vm.Ops["rot3"], vm.Ops["rot"], vm.Ops["rot3"], vm.Ops["rot"], vm.Ops["rot3"], vm.Ops["rot"], vm.Ops["rot3"], vm.Ops["rot"], vm.Ops["rot3"], vm.Ops["rot"], vm.Ops["rot3"]})
	crm := cr1.Mutate(NewRNG(3), &Conf{
		TournamentRatio: 0,
		TournamentP:     0,
		CrossoverMutP:   1,
		PointMutP:       0,
		TransposeMutP:   0,
		CrossoverRatio:  0,
		ErrThreshold:    0,
	})
	t.Log(crm.String())
}
