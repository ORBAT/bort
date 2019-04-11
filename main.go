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
	arg =strings.ReplaceAll(arg, " ", "")
	numStrings := strings.Split(arg, ",")
	nums := make([]int, 0, len(numStrings))
	for _, str := range numStrings {
		n, err := strconv.Atoi(str)
		if err != nil {
			panic(err)
		}
		nums = append(nums, n)
	}

	probs := life.Conf{
		CrossoverRatio: 0.70,
		CrossoverMutP: 0.01,
		PointMutP: 0.08,
		TransposeMutP: 0.08,
		TournamentP: 0.65,
		TournamentRatio: 0.15,
		ErrThreshold: 0.3,
	}

	p := life.NewPopulation(300, vm.MaxExecStackSize, life.NewRNG(0))
	p.DoYourThing(probs, life.SortErrorGen(7, 10, true, life.NewRNG(0)), life.NewRNG(0), 1000, nums, true)
}