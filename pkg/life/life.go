package life

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/ORBAT/bort/pkg/config"
	"github.com/ORBAT/bort/pkg/fucking"
	"github.com/ORBAT/bort/pkg/vm"
)

const MaxError = 1.0

type Genome []vm.Op

type Vector []float64

// EuclidDist returns the "normalized" Euclidean distance between v and other. As both vectors
// should have values between [0,1], the maximum Euclidean distance between them is sqrt(longLen)
// (i.e. square of the dimension), dividing the distance by the sqrt normalizes
func (v Vector) EuclidDist(other Vector) float64 {
	lenV := len(v)
	lenOther := len(other)
	var (
		longer, shorter   = other, v
		longLen, shortLen = lenOther, lenV
	)
	if lenV > lenOther {
		longer, shorter = v, other
		longLen, shortLen = lenV, lenOther
	}
	sumSqDiffs := 0.0
	for longerIdx, longerVal := range longer {
		var shortVal float64
		if longerIdx < shortLen {
			shortVal = shorter[longerIdx]
		}
		diff := longerVal - shortVal
		sumSqDiffs += diff * diff
	}
	return math.Sqrt(sumSqDiffs) / math.Sqrt(float64(longLen))
}

func (g Genome) ToVector() Vector {
	v := make(Vector, 0, len(g))
	for _, op := range g {
		v = append(v, opValues[op.Name])
	}
	return v
}

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
	Genome
	*vm.CPU
	Error float64 // error for this Critter. Lower is better
	ID    string
}

func (c Critter) GoString() string {
	return fmt.Sprintf("<Critter %s Error=%.3f \nCPU %p %+v>\n", c.ID, c.Error, c.CPU, c.CPU)
}

func (c Critter) String() string {
	return fmt.Sprintf("<Critter %s Error=%.3f Genome=\n%s>\n", c.ID, c.Error, c.Genome.String())
}

// Mutate a critter. xoverP gives the probability of crossover mutation, pointMutP for point
// mutation and transposeMutP for transposition.
func (c Critter) Mutate(rng *rand.Rand, cfg *config.Options) Critter {
	if rng.Float64() < cfg.CrossoverMutP {
		cg := CritterGenerator(cfg, rng)
		return c.Cross(cg(), rng, cfg)
	}
	opGen := OpGenerator(rng)
	genomeLen := len(c.Genome)
	newGen := make([]vm.Op, genomeLen)
	copy(newGen, c.Genome)
	for i := range newGen {
		if rng.Float64() < cfg.PointMutP {
			newGen[i] = opGen()
		}

		if rng.Float64() < cfg.TransposeMutP {
			otherPos := i
			for otherPos == i {
				otherPos = rng.Intn(genomeLen)
			}
			newGen[i], newGen[otherPos] = newGen[otherPos], newGen[i]
		}
	}
	return NewCritter(newGen, cfg)
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

func (c Critter) crossSimple(other Critter, rng *rand.Rand, cfg *config.Options, tries int) (offspring Critter) {
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
	if len(offsgen) < 3 || len(offsgen) > cfg.MaxExecStackSize {
		return c.crossSimple(other, rng, cfg, tries+1)
	}
	return NewCritter(offsgen, cfg)
	// alen 5, ap 2
	// a: aa bb cc dd ee
	// [aa bb]
	// blen 3, bp 1
	// b: ff gg hh
}

// 3-way cross
func (c Critter) cross(other Critter, randGen *rand.Rand, tries int, cfg *config.Options) (offspring Critter) {
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

	if offl := len(offspringGenome); offl < 2 || offl > cfg.MaxExecStackSize {
		return c.cross(other, randGen, tries+1, cfg)
	}
	return NewCritter(offspringGenome, cfg).Mutate(randGen, cfg)
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
func (c Critter) Cross(other Critter, rng *rand.Rand, cfg *config.Options) (offspring Critter) {
	offspring = c.cross(other, rng, 0, cfg)
	if len(offspring.Genome) > cfg.MaxExecStackSize {
		offspring.Genome = offspring.Genome[:cfg.MaxExecStackSize]
	}
	return offspring
}

type CritterGen func() Critter

func CritterGenerator(cfg *config.Options, rng *rand.Rand) CritterGen {
	opGen := OpGenerator(rng)
	return func() Critter {
		nOps := 0
		for nOps < 2 {
			nOps = rng.Intn(cfg.MaxExecStackSize-1) + 1
		}
		ops := make([]vm.Op, nOps)
		for i := range ops {
			ops[i] = opGen()
		}
		return NewCritter(ops, cfg)
	}
}

func NewCritter(ops []vm.Op, cfg *config.Options) Critter {
	return Critter{ops, vm.NewCPU(ops, cfg.CPU), MaxError, fmt.Sprintf("%p", &ops)}
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

type ErrorFunction func(c Critter, input ...int) float64

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

func (p Population) SelectFar(rng *rand.Rand, cfg *config.Options, orig Critter) (other Critter, indexInP int) {
	origv := orig.ToVector()
	maxDist := 0.0
	var furthest Critter
	for tries := 0; tries <= 20; tries++ {
		other, indexInP = p.Select(rng, cfg)
		dist := other.ToVector().EuclidDist(origv)
		if dist < cfg.MinEuclDist {
			furthest = other
			break
		}
		if dist > maxDist {
			furthest = other
			maxDist = dist
		}
	}
	return furthest, indexInP
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
func (p Population) Select(rng *rand.Rand, cfg *config.Options) (cr Critter, indexInP int) {
	popSize := len(p)

	tournSize := int(float64(popSize) * cfg.TournamentRatio)

	tournament := make(Population, tournSize)

	// idx is effectively a slice of randomized indices into p, where each index is in idxs once
	idxs := rng.Perm(popSize)
	// pick tournSize individuals into the tournament using random indices
	for i := range tournament {
		tournament[i] = p[idxs[i]]
	}

	sort.Sort(tournament)

	winner := tournament[len(tournament)-1]
	oneLessPp := 1 - cfg.TournamentP
	i := 0
	for ; i < tournSize; i++ {
		if rng.Float64() < (cfg.TournamentP * math.Pow(oneLessPp, float64(i))) {
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
func (p Population) Mutate(rng *rand.Rand, cfg *config.Options) Population {
	nToMutate := cfg.NToMutate(len(p))
	newP := make(Population, nToMutate-1, nToMutate)
	picked := make([]int, 0, nToMutate)
	for i := range newP {
		critter, idx := p.SelectRandom(rng)
		for isIn(picked, idx) {
			critter, idx = p.SelectRandom(rng)
		}
		picked = append(picked, idx)
		newP[i] = critter.Mutate(rng, cfg)
	}

	// elite selection
	newP = append(newP, p.Best())
	return newP
}

// Cross over a part of the population, and return a population with descendants only
func (p Population) Cross(rng *rand.Rand, cfg *config.Options) Population {
	newP := make(Population, cfg.NToCrossover(len(p)))
	for i := range newP {
		var (
			critter1, critter2 Critter
			idx1, idx2         int
		)
		for idx1 == idx2 {
			critter1, idx1 = p.Select(rng, cfg)
			// critter2, idx2 = p.Select(rng, cfg)
			critter2, idx2 = p.SelectFar(rng, cfg, critter1)
		}
		newP[i] = critter1.Cross(critter2, rng, cfg)
	}
	return newP
}

type Stats struct {
	AvgErr, AvgNSteps float64
	LowErr            Population
}

func (p *Population) Stats(errThreshold float64) Stats {
	popSize := float64(len(*p))
	errSum := 0.0
	nStepSum := 0.0
	lowErr := Population{}
	for _, cr := range *p {
		errSum += cr.Error
		nStepSum += float64(cr.NSteps)
		if cr.Error < errThreshold {
			lowErr = append(lowErr, cr)
		}
	}
	return Stats{errSum / popSize, nStepSum / popSize, lowErr}
}

func (p *Population) DoYourThing(cfg *config.Options, errorFn ErrorFunction, rng *rand.Rand, toSort []int) (pop Population, best Critter, bestSort []interface{}) {
	generation := 0
	bestToSortErr := MaxError
	wantSorted := make([]int, len(toSort))
	copy(wantSorted, toSort)
	sort.Ints(wantSorted)
	for ; generation < cfg.MaxGenerations; generation++ {
		p.CalcErrors(errorFn)
		st := p.Stats(cfg.ErrThreshold)

		if cfg.Verbose {
			if generation%100 == 0 {
				genBest := p.Best()
				origInp := genBest.OrigInput()
				want := make([]int, len(origInp))
				copy(want, origInp)
				sort.Ints(want)
				log.Printf("gen %4d - avgErr %1.3f - err<%1.2f = %d - avgNSteps/inp %2.1f - genBest %s err %.3f.\norig: %v\ngot:  %v\nwant: %v\n%s\n",
					generation, st.AvgErr, cfg.ErrThreshold, len(st.LowErr), st.AvgNSteps/float64(len(origInp)), genBest.ID, genBest.Error,
					origInp, genBest.Int, want, genBest.String())
			}
		}

		if candidates := st.LowErr; len(candidates) != 0 {
			for _, candidate := range candidates {
				toSortErr := errorFn(candidate, toSort...)
				if toSortErr < bestToSortErr {
					if cfg.Verbose {
						log.Printf("gen %4d - best sort of your array so far (error %1.3f) :\norig: %v\nnow:  %v\nwant: %v\n%s", generation, toSortErr, toSort, candidate.Int, wantSorted, candidate.String())
					}
					bestToSortErr = toSortErr
					best = candidate
					bestSort = candidate.Int
				}
				if toSortErr == 0 {
					goto otog
				}

			}
		}

		newPop := p.Mutate(rng, cfg)
		newPop = append(newPop, p.Cross(rng, cfg)...)
		*p = newPop
	}
otog:
	return *p, best, bestSort
}

func NewRNG(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(maybeUnixNano(seed)))
}

func OpGenerator(rng *rand.Rand) func() vm.Op {
	return func() vm.Op {
		return vm.Ops[opNames[rng.Intn(len(opNames))]]
	}
}

func NewPopulation(cfg *config.Options, rng *rand.Rand) Population {
	cg := CritterGenerator(cfg, rng)
	p := make(Population, cfg.PopSize)
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

func SortErrorGen(rng *rand.Rand, cfg *config.Options) ErrorFunction {
	fatalErrs := cfg.FatalErrors
	minLen := cfg.MinTrainingArrayLen
	sizeRange := 1 + (cfg.MaxTrainingArrayLen - minLen)
	return func(c Critter, input ...int) float64 {
		inpLen := rng.Intn(sizeRange) + minLen
		var inp, want []int
		if len(input) == 0 {
			inp, want = genTestSlice(inpLen, rng)
		} else {
			inp = input
			want = make([]int, len(input))
			copy(want, input)
			sort.Ints(want)
		}

		_, err := c.Input(inp).Run(fatalErrs)
		if err != nil {
			return MaxError
		}

		outp := fucking.IntSlice(c.Int)

		if isSame(outp, inp) {
			return MaxError
		}

		if isSame(outp, want) {
			return 0
		}

		//  float64(levenshtein(outp, want)) / float64(max(outLen, inpLen))
		// positionalError only works if len(want)==len(outp)
		return positionalError(want, outp)
	}
}

func genTestSlice(inpLen int, rng *rand.Rand) (inp []int, want []int) {
	inp = make([]int, inpLen)
	for i := range inp {
		inp[i] = rng.Intn(21)
	}
	want = make([]int, inpLen)
	copy(want, inp)
	sort.Ints(want)
	if positionalError(inp, want) < 0.5 {
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
var opValues map[string]float64 // op name -> normalized value for that op, idx/(len(opNames))

func init() {
	nOps := len(vm.Ops)
	opNames = make([]string, 0, nOps)
	for name := range vm.Ops {
		opNames = append(opNames, name)
	}
	sort.Strings(opNames)
	opValues = make(map[string]float64, nOps)
	for i, name := range opNames {
		opValues[name] = float64(i) / float64(nOps)
	}
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
}

// WHY DO I NEED TO FUCKING DO THIS MANUALLY GODDAMNIT IT'S 2019
func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

// closestIdx finds the index of needle in haystack that's closest to wantIdx
func closestIdx(needle, wantIdx int, haystack []int) int {
	lenHays := len(haystack)
	for i := 0; i < lenHays; i++ {
		leftIdx := abs(wantIdx-i) % lenHays
		left := haystack[leftIdx]
		if left == needle {
			return leftIdx
		}
		rightIdx := (wantIdx + i) % lenHays
		if rightIdx == leftIdx {
			continue
		}
		right := haystack[rightIdx]
		if right == needle {
			return rightIdx
		}
	}
	return -1
}

// TODO: unfuck. There has to be a better way to do this
func maxDist(len int) float64 {
	if len == 1 {
		return 0
	}

	if len == 2 {
		return 2
	}

	if len == 3 {
		return 4
	}

	return float64((len-1)*2) + maxDist(len-2)
}

// positionalError calculates how different "got" is from "want", assuming that "got" is a permutation of
// "want" (i.e. has exactly the same elements, just in a different order.) It returns a value in the
// range [0,1], so that a result of 0 means that "got" and "want" are identical (each element is shifted
// by 0 positions), and 1 means that each element is as far away from its intended position as
// possible.
//
// It does this by looking at each element in "got", and seeing how far it is from its wanted position.
// As an example, if "want" is
// 	[1 2 3 4]
// and "got" is
//  [4 3 2 1]
// the "4" at index 0 is 3 positions away from its real place (as is the "1" at index 3), the "3" at
// index 1 is 1 position away (and so is "2"). This slice is also the "most wrong" permutation of
// "want", as each element is in the wrong place.
//
// Now, this means that for a slice of length 4, the maximum sum of errors (i.e. the sum of how far
// away each element is from the right spot) is always going to be at most 8; two elements can be 3
// positions away, then the last two can be at most 1 away or in the right place (as there's only
// two positions left for them to fill). This "maximum error" is used to normalize the sum of errors of each element
func positionalError(want, got []int) float64 {
	lenWant := len(want)
	if lenWant != len(got) {
		panic("this shit only works if want and got are the same length")
	}

	errSum := 0.0

	// max errors
	// length 2: 1 2  <- max err 1*2 = 2, because if one element is in the wrong place, both are.
	// length 3: 1 2 3 <- max err 2*2 = 4. Two elements can be at most 2 spots from the right place, and
	//                    the last one has no choice
	// length 4: 1 2 3 4 <- max err 3*2 + 2 = 8 (3*2 plus the max error of length 2)
	// length 5: 1 2 3 4 5 <- 4*2 + 4 = 12 (4*2 plus max err of length 3)
	// length 6: 1 2 3 4 5 6 <- 5*2 + 8 = 18
	// length 7: 1 2 3 4 5 6 7 <- 6*2 + 12 = 24
	// length 8: 1 2 3 4 5 6 7 8 <- 7*2 + 18 = 32

	for wantIdx := 0; wantIdx < lenWant; wantIdx++ {
		errSum += math.Abs(float64(closestIdx(want[wantIdx], wantIdx, got) - wantIdx))
	}
	return errSum / maxDist(lenWant)

	// for wantIdx, wanted := range want {
	// 	errSum += math.Abs(float64(closestIdx(wanted, wantIdx, got) - wantIdx))
	// }

	// return errSum / float64(maxDist(lenWant))
}

// based on https://github.com/agnivade/levenshtein/blob/master/levenshtein.go
func levenshtein(s1, s2 []int) float64 {
	if len(s1) == 0 {
		return 1
	}

	if len(s2) == 0 {
		return 1
	}

	if isSame(s1, s2) {
		return 0
	}

	lenS1 := len(s1)
	lenS2 := len(s2)

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
	return float64(x[lenS1]) / float64(max(lenS1, lenS2))
}
