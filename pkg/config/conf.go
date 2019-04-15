package config

import (
	"math/rand"
)

type CPU struct {
	MaxExecStackSize int     `usage:"the maximum size of the exec stack. If set to 0, will be set to critterSize"`
	MaxStepsPerInput float64 `usage:"governs how many steps per each input item each individual can run. For example, for an input of length 5 and MaxStepsPerInput of 4, each individual would have a total of 20 steps to do its thing"`
	FatalErrors      bool    `usage:"whether errors during execution (such as popping an empty stack) are fatal"`
}

type Stats struct {
	AvgGenerations int `usage:"when generating stats, average over this many generations"`
}

type Options struct {
	// The ratio of the population in a tournament, i.e. tournament size. The smaller this is, the
	// likelier it is that less fit individuals will get to reproduce
	TournamentRatio float64 `usage:"The ratio of the population in a tournament, i.e. tournament size. The smaller this is, the likelier it is that less fit individuals will get to reproduce. Pass 0 to default to a fraction of the population that gives 7 tournament participants"`

	// The likelihood that the best individual in a tournament will win
	TournamentP float64 `usage:"The likelihood that the best individual in a tournament will win"`

	// Probability of crossover mutation
	CrossoverMutP float64 `usage:"Probability of crossover mutation"`
	// Probability of one operation being mutated
	PointMutP float64 `usage:"Probability of one operation being mutated"`
	// Probability of transposition mutation
	TransposeMutP float64 `usage:"Probability of transposition mutation"`

	// Percentage of a population that might be mutated after every generation
	MutationRatio float64 `usage:"Percentage of a population that might be mutated after every generation"`

	MutSigmaRatio float64 `usage:"Mutation probabilities' standard deviation is initially set at probability*MutSigmaRatio, so if MutSigmaRatio=0.1, 	each probability's σ will be 10%, meaning that ~68% of the time the probability's value will be within 10% of the initial value"`

	// ErrThreshold is the error under which the critter will be used to try and solve the input problem
	ErrThreshold float64 `usage:"the error under which the critter will be used to try and solve the input problem"`

	// MinEuclDist is the smallest Euclidean distance to a partner that Select will allow (if at all
	// possible)
	MinEuclDist float64 `usage:"the smallest Euclidean distance to a partner that selection during reproduction will allow (if at all possible)"`

	// MaxGenerations is the maximum number of generations to run
	MaxGenerations int `usage:"the maximum number of generations to run"`

	MinTrainArrLen int `usage:"minimum training array size"`
	MaxTrainArrLen int `usage:"maximum training array size"`

	GlobalMutation bool `usage:"whether to mutate mutationRatio of the population after each generation"`

	PopSize int `usage:"Population size"`

	Verbose bool `usage:"log spam"`

	CritterSize int `usage:"critter size. If 0, will be random between 3 and CPU.maxExecStackSize"`

	CPU
	Stats
}

func (o *Options) NormalP(mutP float64, rng *rand.Rand) float64 {
	return rng.NormFloat64()*(o.MutSigmaRatio*mutP) + mutP
}

// NToMutate returns the number of individuals to mutate in p
func (o *Options) NToMutate(rng *rand.Rand) int {
	return max(min(int(o.NormalP(o.MutationRatio, rng) * float64(o.PopSize)), o.PopSize), 1)
	// return int(math.Floor(o.MutationRatio * float64(o.PopSize)))
}

func (o *Options) MaxCritterSize() int {
	if cs := o.CritterSize; cs == 0 {
		return o.MaxExecStackSize
	} else {
		return cs
	}
}

func (o *Options) SetDefaults() {
	if o.TournamentRatio == 0 {
		o.TournamentRatio = 7 / float64(o.PopSize)
	}

	if o.MaxExecStackSize == 0 {
		o.MaxExecStackSize = o.CritterSize
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}