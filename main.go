package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"unsafe"

	"github.com/ORBAT/bort/pkg/life"
	"github.com/ORBAT/bort/pkg/vm"
)

// var verbose = flag.Bool("verbose", false, "Log spam")
var popSz = flag.Uint("popsize", 500, "Population size")

func flagStruct(s interface{}) {
	sVal := reflect.ValueOf(s).Elem()
	sType := sVal.Type()
	for i := 0; i < sType.NumField(); i++ {
		typeField := sType.Field(i)
		name := typeField.Name
		name = strings.ToLower(name[:1]) + name[1:]
		valField := sVal.Field(i)
		valPtr := unsafe.Pointer(valField.Addr().Pointer())
		fieldDescr := typeField.Tag.Get("usage")
		if fieldDescr == "" {
			fieldDescr = typeField.Name
		}
		switch t := typeField.Type.Kind(); t {
		case reflect.Float64:
			flag.Float64Var((*float64)(valPtr), name, *(*float64)(valPtr), fieldDescr)
		case reflect.Bool:
			flag.BoolVar((*bool)(valPtr), name, *(*bool)(valPtr), fieldDescr)
		default:
			log.Fatalf("I can't deal with fields of type %s", t.String())
		}
	}
}

func main() {
	conf := &life.Conf{
		CrossoverRatio:  0.90,
		CrossoverMutP:   0.01,
		PointMutP:       0.01,
		TransposeMutP:   0.01,
		TournamentP:     0.75,
		TournamentRatio: 2.0 / 500.0,
		ErrThreshold:    0.4,
		MinEuclDist:     0.9,
		Verbose:         false,
	}
	flagStruct(conf)

	flag.Parse()
	if len(flag.Args()) != 0 {
		panic("the first and only non-flag argument must be a comma-separated list of integers")
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

	p := life.NewPopulation(int(*popSz), vm.MaxExecStackSize, life.NewRNG(0))
	errorFn := life.SortErrorGen(5, 25, true, life.NewRNG(0))
	_, _, sortaSorted := p.DoYourThing(conf, errorFn, life.NewRNG(0), 2000, nums)
	fmt.Printf("%v", sortaSorted)
}
