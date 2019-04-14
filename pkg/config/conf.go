package config

import (
	"math"
)

type CPU struct {
	MaxExecStackSize int     `usage:"the maximum size of the exec stack"`
	MaxStepsPerInput float64 `usage:"governs how many steps per each input item each individual can run. For example, for an input of length 5 and MaxStepsPerInput of 4, each individual would have a total of 20 steps to do its thing"`
	FatalErrors      bool    `usage:"whether errors during execution (such as popping an empty stack) are fatal"`
}

type Options struct {
	// The ratio of the population in a tournament, i.e. tournament size. The smaller this is, the
	// likelier it is that less fit individuals will get to reproduce
	TournamentRatio float64 `usage:"The ratio of the population in a tournament, i.e. tournament size. The smaller this is, the likelier it is that less fit individuals will get to reproduce. Pass 0 to default to a fraction of the population that gives 2 tournament participants"`

	// The likelihood that the best individual in a tournament will win
	TournamentP float64 `usage:"The likelihood that the best individual in a tournament will win"`

	// Probability of crossover mutation
	CrossoverMutP float64 `usage:"Probability of crossover mutation"`
	// Probability of one operation being mutated
	PointMutP float64 `usage:"Probability of one operation being mutated"`
	// Probability of transposition mutation
	TransposeMutP float64 `usage:"Probability of transposition mutation"`

	// Percentage of a new population that is generated with crossover mating (the rest are
	// generated with mutation)
	CrossoverRatio float64 `usage:"Percentage of a new population that is generated with crossover mating (the rest are generated with mutation)"`

	// ErrThreshold is the error under which the critter will be used to try and solve the input problem
	ErrThreshold float64 `usage:"the error under which the critter will be used to try and solve the input problem"`

	// MinEuclDist is the smallest Euclidean distance to a partner that Select will allow (if at all
	// possible)
	MinEuclDist float64 `usage:"the smallest Euclidean distance to a partner that selection during reproduction will allow (if at all possible)"`

	// MaxGenerations is the maximum number of generations to run
	MaxGenerations int `usage:"the maximum number of generations to run"`

	MinTrainingArrayLen int `usage:"minimum training array size"`
	MaxTrainingArrayLen int `usage:"maximum training array size"`

	PopSize int `usage:"Population size"`

	Verbose bool `usage:"log spam"`

	CPU
}

// MutationRatio is a convenience method for 1 - ps.CrossoverRatio
func (l *Options) MutationRatio() float64 {
	return 1 - l.CrossoverRatio
}

// NToMutate returns the number of individuals to mutate in p
func (l *Options) NToMutate(popSize int) int {
	return int(math.Floor(l.MutationRatio() * float64(popSize)))
}

// NToCrossover returns the number of individuals to cross over in p
func (l *Options) NToCrossover(popSize int) int {
	return int(math.Ceil(l.CrossoverRatio * float64(popSize)))
}

