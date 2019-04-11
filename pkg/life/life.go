package life

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/ORBAT/bort/pkg/fucking"
	"github.com/ORBAT/bort/pkg/vm"
)

const MaxError = 1

type Genome []vm.Op

func (g Genome) String() string {
	if len(g) == 0 {
		panic("empty genome?")
	}
	var b strings.Builder
	for _, op := range g {
		b.WriteString(op.Name)
		b.WriteRune('\n')
	}
	return b.String()
}

type Critter struct {
	Genome Genome
	*vm.CPU
	Error float64 // error for this Critter. Lower is better
	ID    string
}

func (c Critter) String() string {
	return fmt.Sprintf("<Critter %s Error=%.3f\n%s>", c.ID, c.Error, c.Genome.String())
}

// Mutate a critter. xoverP gives the probability of crossover mutation, pointMutP for point
// mutation and transposeMutP for transposition.
func (c Critter) Mutate(rng *rand.Rand, xoverP, pointMutP, transposeMutP float64) Critter {
	if rng.Float64() < xoverP {
		cg := CritterGenerator(vm.MaxExecStackSize, rng)
		return c.Cross(cg(), rng)
	}
	opGen := OpGenerator(rng)
	genomeLen := len(c.Genome)
	newGen := make([]vm.Op, genomeLen)
	copy(newGen, c.Genome)
	for i := range newGen {
		if rng.Float64() < pointMutP {
			newGen[i] = opGen()
		}

		if rng.Float64() < transposeMutP {
			otherPos := i
			for otherPos == i {
				otherPos = rng.Intn(genomeLen)
			}
			newGen[i], newGen[otherPos] = newGen[otherPos], newGen[i]
		}
	}
	return NewCritter(newGen)
}

func minMax(a, b int) (min, max int) {
	if a < b {
		return a, b
	}
	return b, a
}

// crossPoints returns two points that can be used to slice
// gs, so that a is less than and not equal to b, and b < len(gs)
func (c Critter) crossPoints(randGen *rand.Rand) (a int, b int) {
	lgs := len(c.Genome)
	a = randGen.Intn(lgs)
	b = randGen.Intn(lgs)
	if a == b {
		return c.crossPoints(randGen)
	}
	return minMax(a, b)
}

func (c Critter) crossSimple(other Critter, rng *rand.Rand, tries int) (offspring Critter) {
	if tries > 4 {
		return c
	}
	var a, b Critter
	if rng.Intn(2) == 1 {
		a, b = c, other
	} else {
		a, b = other, c
	}
	alen := len(a.Genome)
	blen := len(b.Genome)

	ap := rng.Intn(alen)
	bp := rng.Intn(blen)
	offsgen := make([]vm.Op, 0, (alen-ap)+(blen-bp))
	offsgen = append(offsgen, a.Genome[:ap]...)
	offsgen = append(offsgen, b.Genome[bp:]...)
	if len(offsgen) < 3 || len(offsgen) > vm.MaxExecStackSize {
		return c.crossSimple(other, rng, tries+1)
	}
	return NewCritter(offsgen)
	// alen 5, ap 2
	// a: aa bb cc dd ee
	// [aa bb]
	// blen 3, bp 1
	// b: ff gg hh
}

// 3-way cross
func (c Critter) cross(other Critter, randGen *rand.Rand, tries int) (offspring Critter) {
	if tries > 4 {
		return c
	}
	var a, b Critter
	if randGen.Intn(2) == 1 {
		a, b = c, other
	} else {
		a, b = other, c
	}

	alen := len(a.Genome)

	aMinPt, aMaxPt := a.crossPoints(randGen)
	bMinPt, bMaxPt := b.crossPoints(randGen)

	// this length calculation is probably bullshit but meh
	offspringGenome := make([]vm.Op, 0, aMinPt+(bMaxPt-bMinPt)+(alen-aMaxPt))
	offspringGenome = append(offspringGenome, a.Genome[:aMinPt+1]...)
	offspringGenome = append(offspringGenome, b.Genome[bMinPt:bMaxPt+1]...)
	offspringGenome = append(offspringGenome, a.Genome[aMaxPt:]...)

	if offl := len(offspringGenome); offl < 2 || offl > vm.MaxExecStackSize {
		return c.cross(other, randGen, tries+1)
	}
	return NewCritter(offspringGenome)
}

// Cross crosses c with other using two crossover points, producing one offspring genome.
// The operation can be visualized as follows:
//     Parent A
//     0 1 2 3 4 5 6 7 8 9
//         ^       ^
//     Parent B
//     a b c d e f g h i j k l m n
//               ^         ^
//     Offspring
//     0 1 f g h i j 6 7 8 9
func (c Critter) Cross(other Critter, rng *rand.Rand) (offspring Critter) {
	offspring = c.crossSimple(other, rng, 0)
	if len(offspring.Genome) > vm.MaxExecStackSize {
		offspring.Genome = offspring.Genome[:vm.MaxExecStackSize]
	}
	return offspring
}

type CritterGen func() Critter

func CritterGenerator(maxOps int, rng *rand.Rand) CritterGen {
	opGen := OpGenerator(rng)
	return func() Critter {
		nOps := 0
		for nOps < 2 {
			nOps = rng.Intn(maxOps-1) + 1
		}
		ops := make([]vm.Op, nOps)
		for i := range ops {
			ops[i] = opGen()
		}
		return NewCritter(ops)
	}
}

func NewCritter(ops []vm.Op) Critter {
	return Critter{ops, vm.NewCPU(ops), MaxError, fmt.Sprintf("%p", &ops)}
}

type Population []Critter

func (p Population) Len() int {
	return len(p)
}

func (p Population) Less(i, j int) bool {
	return p[i].Error < p[j].Error
}

func (p Population) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

type ErrorFunction func(Critter) float64

func (p Population) CalcErrors(errorFn ErrorFunction) Population {
	for i, critter := range p {
		critter.Error = errorFn(critter)
		p[i] = critter
	}
	return p
}

func (p Population) Sort() Population {
	sort.Sort(p)
	return p
}

func (p Population) Best() Critter {
	sort.Sort(p)
	return p[0]
}

// Delete individual at index idx
func (p *Population) Delete(idx int) {
	pp := *p
	copy((pp)[idx:], (pp)[idx+1:])
	last := len(pp) - 1
	(pp)[last] = Critter{}
	*p = (pp)[:last]
}

//
// Select an individual from the population using tournament selection.
//
// Tournament selection uses the following algorithm:
//   choose tournamentRatio * len(p) individuals from the population
//   choose the best individual from pool/tournament with probability TournamentP
//   choose the second best individual with probability tournamentP*(1-tournamentP)
//   choose the third best individual with probability tournamentP*((1-tournamentP)^2) etc.
//
// tournamentP = 1 always returns the best individual of the tournament, and a really small
// tournamentRatio (so only eg. 1 individual ends up in the tournament) will make selection
// effectively random
func (p Population) Select(rng *rand.Rand, tournamentRatio, tournamentP float64) (cr Critter, indexInP int) {
	popSize := len(p)

	tournSize := int(float64(popSize) * tournamentRatio)

	tournament := make(Population, tournSize)

	// idx is effectively a slice of randomized indices into p, where each index is in idxs once
	idxs := rng.Perm(popSize)

	// pick tournSize individuals into the tournament using random indices
	for i := range tournament {
		tournament[i] = p[idxs[i]]
	}

	sort.Sort(tournament)

	winner := tournament[len(tournament)-1]
	oneLessPp := 1 - tournamentP
	i := 0
	for ; i < tournSize; i++ {
		if rng.Float64() < (tournamentP * math.Pow(oneLessPp, float64(i))) {
			winner = tournament[i]
			break
		}
	}
	if i == tournSize { // nobody "won" the tournament and for loop ran all the way through
		i -= 1
	}
	return winner, idxs[i]
}

func (p Population) SelectRandom(rng *rand.Rand) (Critter, int) {
	idx := rng.Intn(len(p))
	return p[idx], idx
}

func isIn(ints []int, i int) bool {
	for _, elem := range ints {
		if i == elem {
			return true
		}
	}
	return false
}

// Mutate a part of the population, and return a new population with only mutated critters + the best of p
func (p Population) Mutate(rng *rand.Rand, ps Probabilities) Population {
	nToMutate := ps.NToMutate(p)
	newP := make(Population, /*nToMutate-1,*/ nToMutate)
	picked := make([]int, 0, nToMutate)
	for i := range newP {
		critter, idx := p.SelectRandom(rng)
		for isIn(picked, idx) {
			critter, idx = p.SelectRandom(rng)
		}
		picked = append(picked, idx)
		newP[i] = critter.Mutate(rng, ps.CrossoverMutP, ps.PointMutP, ps.TransposeMutP)
	}
	// elite selection
	// newP = append(newP, p.Best())
	return newP
}

// Cross over a part of the population, and return a population with descendants only
func (p Population) Cross(rng *rand.Rand, ps Probabilities) Population {
	newP := make(Population, ps.NToCrossover(p))
	for i := range newP {
		var (
			critter1, critter2 Critter
			idx1, idx2         int
		)
		for idx1 == idx2 {
			critter1, idx1 = p.Select(rng, ps.TournamentRatio, ps.TournamentP)
			critter2, idx2 = p.Select(rng, ps.TournamentRatio, ps.TournamentP)
		}
		newP[i] = critter1.Cross(critter2, rng)
	}
	return newP
}

type Probabilities struct {
	// The ratio of the population in a tournament, i.e. tournament size. The smaller this is, the
	// likelier it is that less fit individuals will get to reproduce
	TournamentRatio float64
	// The likelihood that the best individual in a tournament will win
	TournamentP float64

	// Probability of crossover mutation
	CrossoverMutP float64
	// Probability of one operation being mutated
	PointMutP float64
	// Probability of transposition mutation
	TransposeMutP float64

	// Percentage of a new population that is generated with crossover mating (the rest are
	// generated with mutation)
	CrossoverRatio float64
}

// MutationRatio is a convenience method for 1 - ps.CrossoverRatio
func (ps Probabilities) MutationRatio() float64 {
	return 1 - ps.CrossoverRatio
}

// NToMutate returns the number of individuals to mutate in p
func (ps Probabilities) NToMutate(p Population) int {
	return int(math.Floor(ps.MutationRatio() * float64(len(p))))
}

// NToCrossover returns the number of individuals to cross over in p
func (ps Probabilities) NToCrossover(p Population) int {
	return int(math.Ceil(ps.CrossoverRatio * float64(len(p))))
}

type Stats struct {
	AvgErr  float64
	ZeroErr Population
}

func (p *Population) Stats() Stats {
	popSize := float64(len(*p))
	sum := 0.0
	zeroErr := Population{}
	for _, cr := range *p {
		sum += cr.Error
		if cr.Error == 0 {
			zeroErr = append(zeroErr, cr)
		}
	}
	return Stats{sum / popSize, zeroErr}
}

func (p *Population) DoYourThing(ps Probabilities, errorFn ErrorFunction, rng *rand.Rand, maxGen int, toSort []int, ignoreErrs bool) (pop Population, best Critter) {
	generation := 0
	wantToSort := make([]int, len(toSort))
	copy(wantToSort, toSort)
	sort.Ints(wantToSort)
	for ; generation < maxGen; generation++ {
		p.CalcErrors(errorFn)
		st := p.Stats()

		if generation%1000 == 0 {
			genBest := p.Best()
			origInp := genBest.OrigInput()
			want := make([]int, len(origInp))
			copy(want, origInp)
			sort.Ints(want)
			log.Printf("gen %5d - avg %.3f - no err %2d - genBest %s err %.3f.\norig: %v\ngot:  %v\nwant: %v\n", generation, st.AvgErr, len(st.ZeroErr), genBest.ID, genBest.Error,
				origInp, genBest.Int, want)
		}

		if zeros := st.ZeroErr; len(zeros) != 0 {
			for _, critter := range zeros {
				critter.Input(toSort).Run(ignoreErrs)
				if isSame(wantToSort, fucking.IntSlice(critter.Int)) {
					log.Printf("gen %4d - critter %s sorted %v -> %v\nsource:\n%s", generation, critter.ID, toSort, wantToSort, critter.String())
					return *p, critter
				}
			}
		}

		newPop := p.Mutate(rng, ps)
		newPop = append(newPop, p.Cross(rng, ps)...)
		*p = newPop
	}

	return *p, best
}

func NewRNG(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(maybeUnixNano(seed)))
}

func OpGenerator(rng *rand.Rand) func() vm.Op {
	return func() vm.Op {
		return vm.Ops[opNames[rng.Intn(len(opNames))]]
	}
}

func NewPopulation(popSize, maxOps int, rng *rand.Rand) Population {
	cg := CritterGenerator(maxOps, rng)
	p := make(Population, popSize)
	for i := range p {
		p[i] = cg()
	}
	return p
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

func SortErrorGen(minSize, maxSize int, ignoreStepErrs bool, rng *rand.Rand) ErrorFunction {
	return func(c Critter) float64 {
		inpLen := rng.Intn(1+(maxSize-minSize)) + minSize

		inp, want := genTestSlice(inpLen, rng)
		_, err := c.Input(inp).Run(ignoreStepErrs)
		if err != nil {
			return MaxError
		}
		outp := fucking.IntSlice(c.Int)

		outLen := len(outp)

		if inpLen != outLen {
			return MaxError
		}

		if isSame(outp, inp) {
			return MaxError
		}

		if isSame(outp, want) {
			return 0
		}

		return float64(levenshtein(outp, want)) / float64(max(outLen, inpLen))

		// outLen := len(outp)
		// if outLen != inpLen {
		// 	return MaxError
		// }
		//
		// errCount := 0
		// for i, out := range outp {
		// 	if want[i] != out {
		// 		errCount++
		// 	}
		// }
		//
		// errv := float64(errCount) / float64(inpLen)
		// return errv
	}
}

func genTestSlice(inpLen int, rng *rand.Rand) (inp []int, want []int) {
	inp = make([]int, inpLen)
	for i := range inp {
		inp[i] = rng.Intn(50)
	}
	want = make([]int, inpLen)
	copy(want, inp)
	sort.Ints(want)
	if levenshtein(inp, want) < inpLen/2 {
		return genTestSlice(inpLen, rng)
	}
	return inp, want
}

func maybeUnixNano(seed int64) int64 {
	if seed == 0 {
		return time.Now().UnixNano()
	}
	return seed
}

var opNames []string

func init() {
	opNames = make([]string, 0, len(vm.Ops))
	for name := range vm.Ops {
		opNames = append(opNames, name)
	}
	sort.Strings(opNames)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

func levenshtein(s1, s2 []int) int {
	if len(s1) == 0 {
		return len(s2)
	}

	if len(s2) == 0 {
		return len(s1)
	}

	if isSame(s1, s2) {
		return 0
	}

	// We need to convert to []rune if the strings are non-ascii.
	// This could be avoided by using utf8.RuneCountInString
	// and then doing some juggling with rune indices.
	// The primary challenge is keeping track of the previous rune.
	// With a range loop, its not that easy. And with a for-loop
	// we need to keep track of the inter-rune width using utf8.DecodeRuneInString

	// swap to save some memory O(min(a,b)) instead of O(a)
	if len(s1) > len(s2) {
		s1, s2 = s2, s1
	}

	lenS1 := len(s1)
	lenS2 := len(s2)

	// init the row
	x := make([]int, lenS1+1)
	for i := 0; i < len(x); i++ {
		x[i] = i
	}

	// make a dummy bounds check to prevent the 2 bounds check down below.
	// The one inside the loop is particularly costly.
	_ = x[lenS1]
	// fill in the rest
	for i := 1; i <= lenS2; i++ {
		prev := i
		var current int
		for j := 1; j <= lenS1; j++ {
			if s2[i-1] == s1[j-1] {
				current = x[j-1] // match
			} else {
				current = min(min(x[j-1]+1, prev+1), x[j]+1)
			}
			x[j-1] = prev
			prev = current
		}
		x[lenS1] = prev
	}
	return x[lenS1]
}
