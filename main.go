package main

import (
	"os"
	"strconv"
	"strings"

	"github.com/ORBAT/bort/pkg/life"
	"github.com/ORBAT/bort/pkg/vm"
)

func main() {
	if len(os.Args) == 1 {
		panic("second argument must be comma-separated list of integers")
	}

	arg := os.Args[1]
	arg = strings.ReplaceAll(arg, " ", "")
	numStrings := strings.Split(arg, ",")
	nums := make([]int, 0, len(numStrings))
	for _, str := range numStrings {
		n, err := strconv.Atoi(str)
		if err != nil {
			panic(err)
		}
		nums = append(nums, n)
	}

	conf := &life.Conf{
		CrossoverRatio:  0.85,
		CrossoverMutP:   0.04,
		PointMutP:       0.03,
		TransposeMutP:   0.03,
		TournamentP:     0.65,
		TournamentRatio: 2.0 / 50.0,
		ErrThreshold:    0.35,
		MinEuclDist:     0.7,
	}

	p := life.NewPopulation(600, vm.MaxExecStackSize, life.NewRNG(0))
	p.DoYourThing(conf, life.SortErrorGen(vm.MaxExecStackSize+5, vm.MaxExecStackSize+10, true, life.NewRNG(0)), life.NewRNG(0), 300, nums, true)
}
