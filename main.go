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

	probs := life.Probabilities{
		CrossoverRatio: 0.8,
		CrossoverMutP: 0.01,
		PointMutP: 0.02,
		TransposeMutP: 0.02,
		TournamentP: 0.60,
		TournamentRatio: 0.33,
	}

	p := life.NewPopulation(30, vm.MaxExecStackSize, life.NewRNG(0))
	p.DoYourThing(probs, life.SortErrorGen(10, 15, false, life.NewRNG(0)), life.NewRNG(0), 500000, nums, false)
}