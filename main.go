package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/ORBAT/bort/pkg/config"
	"github.com/ORBAT/bort/pkg/flagon"
	"github.com/ORBAT/bort/pkg/life"
)

func main() {
	conf := &config.Options{
		MutationRatio:   0.05,
		CrossoverMutP:   0.001,
		InitialMutSigmaRatio: 0.01,
		PointMutP:       0.01,
		TransposeMutP:   0.01,
		TournamentP:     0.65,
		TournamentRatio: 0.0,
		ErrThreshold:    0.5,
		MinEuclDist:     0.7,
		MaxGenerations:  10000,
		PopSize:         300,
		Verbose:         false,
		MinTrainArrLen:  5,
		MaxTrainArrLen:  15,
		GlobalMutation:  true,
		CPU: config.CPU{
			MaxStepsPerInput: 15,
			MaxExecStackSize: 30,
			FatalErrors:      false,
		},
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

	conf.SetDefaults()

	p := life.RandPopulation(conf, life.NewRNG(0))
	errorFn := life.SortErrorGen(0, conf)
	_, _, sortaSorted := p.DoYourThing(conf, errorFn, life.NewRNG(0), nums)
	fmt.Printf("%v", sortaSorted)
}
