package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/ORBAT/bort/pkg/life"
	"github.com/ORBAT/bort/pkg/vm"
)

var verbose = flag.Bool("verbose", false, "Log spam")

func main() {
	flag.Parse()
	if len(flag.Args()) == 0 {
		panic("second argument must be comma-separated list of integers")
	}

	log.SetOutput(os.Stderr)
	arg := flag.Args()[0]
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
	const popSz = 500
	conf := &life.Conf{
		CrossoverRatio:  0.90,
		CrossoverMutP:   0.01,
		PointMutP:       0.01,
		TransposeMutP:   0.01,
		TournamentP:     0.75,
		TournamentRatio: 2.0 / popSz,
		ErrThreshold:    0.4,
		MinEuclDist:     0.9,
		Verbose:         *verbose,
	}

	p := life.NewPopulation(popSz, vm.MaxExecStackSize, life.NewRNG(0))
	errorFn := life.SortErrorGen(5, 25, true, life.NewRNG(0))
	_, _, sortaSorted := p.DoYourThing(conf, errorFn, life.NewRNG(0), 2000, nums)
	fmt.Printf("%v", sortaSorted)
}
