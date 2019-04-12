package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/ORBAT/bort/pkg/flagon"
	"github.com/ORBAT/bort/pkg/life"
)

func main() {
	conf := &life.Conf{
		CrossoverRatio:  0.90,
		CrossoverMutP:   0.01,
		PointMutP:       0.007,
		TransposeMutP:   0.007,
		TournamentP:     0.65,
		TournamentRatio: 0.0,
		ErrThreshold:    0.4,
		MinEuclDist:     0.9,
		MaxGenerations:  1000,
		PopSize:         500,
		Verbose:         false,
		MaxExecStackSize: 28,
	}
	flagon.Struct(conf)

	flag.Usage = func() {
		progName := path.Base(os.Args[0])
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s: %s [options] 1,2,3,...\neg: %s -popSize 200 6,5,4,3,2\n\n", progName, progName, progName)
		flag.PrintDefaults()
	}
	flag.Parse()
	if len(flag.Args()) != 1 {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "the first and only non-flag argument must be a comma-separated list of integers\n\n")
		flag.Usage()
		os.Exit(1)
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

	if conf.TournamentRatio == 0 {
		conf.TournamentRatio = 2/float64(conf.PopSize)
	}

	p := life.NewPopulation(conf, life.NewRNG(0))
	errorFn := life.SortErrorGen(5, 25, true, life.NewRNG(0))
	_, _, sortaSorted := p.DoYourThing(conf, errorFn, life.NewRNG(0), nums)
	fmt.Printf("%v", sortaSorted)
}
