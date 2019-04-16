package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ORBAT/bort/pkg/config"
	"github.com/ORBAT/bort/pkg/flagon"
	"github.com/ORBAT/bort/pkg/life"
)

func main() {
	conf := &config.Options{
		MutationRatio:   0.05,
		CrossoverMutP:   0.001,
		MutSigmaRatio:   0.5,
		PointMutP:       0.009,
		TransposeMutP:   0.016,
		TournamentP:     0.60,
		TournamentRatio: 0.0,
		ErrThreshold:    0.6,
		MinEuclDist:     0.8,
		MaxGenerations:  10000,
		PopSize:         440,
		Verbose:         false,
		MinTrainArrLen:  0,
		MaxTrainArrLen:  0,
		GlobalMutation:  true,
		CritterSize:     25,
		CPU: config.CPU{
			MaxStepsPerInput: 15,
			MaxExecStackSize: 35,
			FatalErrors:      false,
		},

		Stats: config.Stats{
			AvgGenerations: 15,
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
	if conf.MinTrainArrLen == 0 {
		conf.MinTrainArrLen = len(nums)
	}

	if conf.MaxTrainArrLen == 0 {
		conf.MaxTrainArrLen = len(nums)
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	p := life.NewPopulation(conf, rng)
	errorFn := life.SortErrorGen(0, conf)
	bestToSortErr := 1.0

	wantSorted := make([]int, len(nums))
	copy(wantSorted, nums)
	sort.Ints(wantSorted)

	var (
		bestSort []interface{}
		best     *life.Critter
	)

	for i := 0; i < conf.MaxGenerations; i++ {
		life.ParsimonyPressure(p.Step(conf, errorFn, rng))
		if candidates := p.Stats.LowErr; len(candidates) != 0 {
			for _, candidate := range candidates {
				toSortErr := errorFn(candidate, nums...)
				if toSortErr < bestToSortErr {
					if conf.Verbose {
						log.Printf("gen %4d - best sort of your array so far (error %1.3f) :\norig: %v\nnow:  %v\nwant: %v\n%s", p.Generation, toSortErr, nums, candidate.Int, wantSorted, candidate.String())
					}
					bestToSortErr = toSortErr
					bestSort = candidate.Int
					best = candidate
				}
				if toSortErr == 0 {
					goto otog
				}

			}
		}
	}
otog:

	if conf.Verbose && best != nil {
		best.CalcError(errorFn, nums...)
		log.Printf("Solution after %d generations: %s\n", p.Generation, best)
	}
	fmt.Printf("%v", bestSort)
}

func isSame(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i, aa := range a {
		if aa != b[i] {
			return false
		}
	}
	return true
}
