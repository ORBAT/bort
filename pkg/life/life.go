package life

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/VividCortex/ewma"

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

func (c *Critter) CalcError(errorFn ErrorFunction, input ...int) *Critter {
	c.Error = errorFn(c, input...)
	return c
}

func (c Critter) GoString() string {
	return fmt.Sprintf("<Critter %s Error=%.3f \nCPU %p %+v>\n", c.ID, c.Error, c.CPU, c.CPU)
}

func (c Critter) String() string {
	return fmt.Sprintf("<Critter %s Error=%.3f Genome=\n%s>\n", c.ID, c.Error, c.Genome.String())
}

// Mutate a critter. xoverP gives the probability of crossover mutation, pointMutP for point
// mutation and transpP for transposition.
func (c Critter) Mutate(rng *rand.Rand, cfg *config.Options, pointP, transpP, xoverMutP float64) *Critter {
	if rng.Float64() < xoverMutP {
		cg := CritterGenerator(cfg, rng)
		crossed, _ := c.Cross(rng, cg(), cfg)
		return crossed
	}

	opGen := OpGenerator(rng)
	genomeLen := len(c.Genome)
	newGen := make([]vm.Op, genomeLen)
	copy(newGen, c.Genome)
	for i := range newGen {
		if rng.Float64() < pointP {
			newGen[i] = opGen()
		}

		if rng.Float64() < transpP {
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
	leng := len(c.Genome)
	a = randGen.Intn(leng)
	b = randGen.Intn(leng)
	if a == b {
		return c.crossPoints(randGen)
	}
	return minMax(a, b)
}

// toPieces  for a genome with len 6, and minPoint=2, maxPoint=5
// a b c d e f
//   ^     ^
// returns [a, b], [c, d, e], [f]
func (c Critter) toPieces(minPoint, maxPoint int) []Genome {
	g := c.Genome
	return []Genome{g[0:minPoint], g[minPoint:maxPoint], g[maxPoint:]}
}

func tooLong(aMin, aMax, aLen, bMin, bMax, bLen, maxLen int) (bool) {
	off1l := aMin + (bMax - bMin) + (aLen - aMax)
	off2l := bMin + (aMax - aMin) + (bLen - bMax)
	return off1l > maxLen || off2l > maxLen
}

// 3-way cross
func (c *Critter) cross(other *Critter, randGen *rand.Rand, tries int, cfg *config.Options) (offspring1, offspring2 *Critter) {
	var a, b *Critter
	if randGen.Intn(2) == 1 {
		a, b = c, other
	} else {
		a, b = other, c
	}

	aLen := len(a.Genome)
	bLen := len(a.Genome)
	var bMinPt, bMaxPt int
	aMinPt, aMaxPt := a.crossPoints(randGen)
	if cfg.CritterSize == 0 {
		bMinPt, bMaxPt = b.crossPoints(randGen)
	} else {
		bMinPt, bMaxPt = aMinPt, aMaxPt
	}
	critSz := cfg.MaxCritterSize()
	for tries := 0; tooLong(aMinPt, aMaxPt, aLen, bMinPt, bMaxPt, bLen, critSz) && tries < 6; tries++ {
		aMinPt, aMaxPt = a.crossPoints(randGen)
		bMinPt, bMaxPt = b.crossPoints(randGen)
	}

	aPieces := a.toPieces(aMinPt, aMaxPt)
	bPieces := b.toPieces(bMinPt, bMaxPt)

	offs1Piece1, offs1Piece2, offs1Piece3 := aPieces[0], bPieces[1], aPieces[2]
	offs2Piece1, offs2Piece2, offs2Piece3 := bPieces[0], aPieces[1], bPieces[2]

	offs1Genome := make(Genome, 0, len(offs1Piece1)+len(offs1Piece2)+len(offs1Piece3))
	offs1Genome = append(append(append(offs1Genome, aPieces[0]...), bPieces[1]...), aPieces[2]...)
	offs2Genome := make(Genome, 0, len(offs2Piece1)+len(offs2Piece2)+len(offs2Piece3))
	offs2Genome = append(append(append(offs2Genome, bPieces[0]...), aPieces[1]...), bPieces[2]...)

	return NewCritter(cutDown(offs1Genome, critSz), cfg),
		NewCritter(cutDown(offs2Genome, critSz), cfg)
}

func cutDown(g Genome, maxLen int) Genome {
	if len(g) < maxLen {
		return g
	}
	return g[:maxLen]
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
func (c *Critter) Cross(rng *rand.Rand, other *Critter, cfg *config.Options) (offspring1, offspring2 *Critter) {
	offspring1, offspring2 = c.cross(other, rng, 0, cfg)
	pointP := cfg.NormalP(cfg.PointMutP, rng)
	transpP := cfg.NormalP(cfg.TransposeMutP, rng)
	xoverP := cfg.NormalP(cfg.CrossoverMutP, rng)
	return offspring1.Mutate(rng, cfg, pointP, transpP, xoverP),
		offspring2.Mutate(rng, cfg, pointP, transpP, xoverP)
}

type CritterGen func() *Critter

func CritterGenerator(cfg *config.Options, rng *rand.Rand) CritterGen {
	opGen := OpGenerator(rng)
	return func() *Critter {
		nOps := 0

		if critSz := cfg.CritterSize; critSz == 0 {
			for nOps < 2 {
				nOps = rng.Intn(cfg.MaxExecStackSize-1) + 1
			}
		} else {
			nOps = critSz
		}

		ops := make([]vm.Op, nOps)
		for i := range ops {
			ops[i] = opGen()
		}
		return NewCritter(ops, cfg)
	}
}

func NewCritter(ops Genome, cfg *config.Options) *Critter {
	return &Critter{ops, vm.NewCPU(ops, cfg.CPU), MaxError, fmt.Sprintf("%p", &ops)}
}

type Population struct {
	Critters
	*Stats
	Size int
}

func NewPopulation(cfg *config.Options, rng *rand.Rand) Population {
	cs := RandCritters(cfg, rng)
	nGen := float64(cfg.Stats.AvgGenerations)
	avgErr := ewma.NewMovingAverage(nGen)
	avgErr.Set(MaxError)
	p := Population{
		cs,
		&Stats{
			AvgErr:                 avgErr,
			AvgStepsPerInp:         ewma.NewMovingAverage(nGen),
			BetterThanParentsRatio: ewma.NewMovingAverage(nGen),
			StepTime:               ewma.NewMovingAverage(nGen),
			AvgNLowErr:             ewma.NewMovingAverage(nGen),
			BestError:              ewma.NewMovingAverage(nGen),
			cfg:                    cfg,
		},
		cfg.PopSize,
	}
	return p
}

func (p *Population) calcStats(errThreshold float64) {
	popSize := float64(p.Len())
	errSum := 0.0
	nStepsPerInpSum := 0.0
	lowErr := Critters{}
	lowestErr := p.Critters[0]
	for _, cr := range p.Critters {
		crErr := cr.Error
		if crErr < lowestErr.Error {
			lowestErr = cr
		}
		errSum += crErr
		if nSteps := cr.NSteps; nSteps != 0 {
			nStepsPerInpSum += float64(nSteps) / float64(cr.InpLen)
		}
		if crErr < errThreshold {
			lowErr = append(lowErr, cr)
		}
	}
	p.Stats.AvgErr.Add(errSum / popSize)
	p.Stats.AvgStepsPerInp.Add(nStepsPerInpSum / popSize)
	p.Stats.LowErr = lowErr
	p.Stats.AvgNLowErr.Add(float64(lowErr.Len()))

	p.Stats.BestError.Add(lowestErr.Error)
}

func (p *Population) Step(cfg *config.Options, errorFn ErrorFunction, rng *rand.Rand) *Population {
	stopStepTimer := timer()
	offs, parents := p.Cross(rng, cfg)
	if cfg.GlobalMutation {
		p.Mutate(rng, cfg)
	}
	p.CalcErrors(errorFn)
	p.calcStats(cfg.ErrThreshold)
	nBetter := offs.BetterThan(parents)
	p.Stats.BetterThanParentsRatio.Add(float64(nBetter / 2))
	p.Generation++
	if cfg.Verbose && p.Generation%300 == 0 {
		genBest := p.Best()
		origInp := genBest.OrigInput()
		want := make([]int, len(origInp))
		copy(want, fucking.IntSlice(origInp))
		sort.Ints(want)
		log.Printf(p.Stats.String()+"\norig: %v\ngot:  %v\nwant: %v\n%s\n",
			origInp, genBest.Int, want, genBest.String())
	}

	const step = 0.05

	if p.BetterThanParentsRatio.Value() > 0.20 {
		cfg.MutSigmaRatio = math.Min(cfg.MutSigmaRatio*1+step, 1000)
	} else {
		cfg.MutSigmaRatio = math.Max(cfg.MutSigmaRatio*1-(step*2), 0.001)
	}

	p.StepTime.Add(float64(stopStepTimer(p.Size)))
	return p
}

type Critters []*Critter

func NewCritters(size int) Critters {
	return make(Critters, size)
}

func (cs Critters) BetterThan(others Critters) (nBetter int) {
outer:
	for _, ourCritter := range cs {
		for _, otherCritter := range others {
			if ourCritter.Error < otherCritter.Error {
				nBetter++
				continue outer
			}
		}
	}
	return nBetter
}

func (cs Critters) Len() int {
	return len(cs)
}

func (cs Critters) Less(i, j int) bool {
	return cs[i].Error < cs[j].Error
}

func (cs Critters) Swap(i, j int) {
	cs[i], cs[j] = cs[j], cs[i]
}

type ErrorFunction func(c *Critter, input ...int) float64

type PopModifier func(p *Population) *Population

type PopModifierChain []PopModifier

func (pm PopModifierChain) RunOn(p *Population) *Population {
	for _, fn := range pm {
		p = fn(p)
	}
	return p
}

func ParsimonyPressure(p *Population) *Population {
	minSteps, maxSteps := math.MaxInt64, math.MinInt64
	for _, cr := range p.Critters {
		if st := cr.NSteps; st < minSteps {
			minSteps = st
		} else if st > maxSteps {
			maxSteps = st
		}
	}

	stepRange := float64(maxSteps - minSteps)

	if stepRange == 0 {
		for _, cr := range p.Critters {
			cr.Error *= 0.75
			cr.Error += 0.25
		}
		return p
	}

	for _, cr := range p.Critters {
		cr.Error *= 0.75
		cr.Error += (float64(cr.NSteps-minSteps) / stepRange) * 0.25
	}

	return p
}

func calcErrWorker(p Critters, errorFn ErrorFunction, wg *sync.WaitGroup, input ...int) {
	for i, critter := range p {
		p[i] = critter.CalcError(errorFn, input...)
	}
	wg.Done()
}

func (cs Critters) Clone() Critters {
	cscopy := make(Critters, len(cs))
	copy(cscopy, cs)
	return cscopy
}

func (cs Critters) CalcErrors(errorFn ErrorFunction, input ...int) Critters {
	numCPU := runtime.NumCPU()
	lencs := len(cs)
	batchSize := (lencs + numCPU - 1) / numCPU
	var wg sync.WaitGroup

	for i := 0; i < lencs; i += batchSize {
		last := i + batchSize
		if last > lencs {
			last = lencs
		}
		wg.Add(1)
		go calcErrWorker(cs[i:last], errorFn, &wg, input...)
	}

	wg.Wait()
	return cs
}

func (cs Critters) Best() *Critter {
	lowestErr := cs[0]
	for _, cr := range cs {
		if cr.Error < lowestErr.Error {
			lowestErr = cr
		}
	}
	return lowestErr
}

// Delete individual at index idx
func (cs *Critters) Delete(idx int) {
	pp := *cs
	copy((pp)[idx:], (pp)[idx+1:])
	last := len(pp) - 1
	(pp)[last] = &Critter{}
	*cs = (pp)[:last]
}

func (cs Critters) SelectFar(rng *rand.Rand, cfg *config.Options, orig *Critter) (other *Critter, indexInP int) {
	origv := orig.ToVector()
	maxDist := 0.0
	var furthest *Critter
	for tries := 0; tries <= 20; tries++ {
		other, indexInP = cs.Select(rng, cfg)
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
// Select an individual from cs using tournament selection.
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
func (cs Critters) Select(rng *rand.Rand, cfg *config.Options) (cr *Critter, indexInP int) {
	popSize := len(cs)

	tournSize := int(float64(popSize) * cfg.TournamentRatio)

	tournament := NewCritters(tournSize)

	// idx is effectively a slice of randomized indices into cs, where each index is in idxs once
	idxs := rng.Perm(popSize)
	// pick tournSize individuals into the tournament using random indices
	for i := range tournament {
		tournament[i] = cs[idxs[i]]
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

func (cs Critters) SelectRandom(rng *rand.Rand) (*Critter, int) {
	idx := rng.Intn(len(cs))
	return cs[idx], idx
}

func isIn(ints []int, i int) bool {
	for _, elem := range ints {
		if i == elem {
			return true
		}
	}
	return false
}

// Mutate some of the critters. Modifies contents of cs
func (cs Critters) Mutate(rng *rand.Rand, cfg *config.Options) Critters {
	nToMutate := cfg.NToMutate(rng)
	picked := make([]int, 0, nToMutate)
	pointP := cfg.NormalP(cfg.PointMutP, rng)
	transpP := cfg.NormalP(cfg.TransposeMutP, rng)
	xoverP := cfg.NormalP(cfg.CrossoverMutP, rng)

	for i := 0; i < nToMutate; i++ {
		critter, idx := cs.SelectRandom(rng)
		for isIn(picked, idx) {
			critter, idx = cs.SelectRandom(rng)
		}
		picked = append(picked, idx)
		cs[i] = critter.Mutate(rng, cfg, pointP, transpP, xoverP)
	}

	return cs
}

// Cross two individuals and replace two random individuals with the offspring
func (cs Critters) Cross(rng *rand.Rand, cfg *config.Options) (offspring, parentsCritters Critters) {
	var (
		critter1, critter2 *Critter
		idx1, idx2         int
	)
	for idx1 == idx2 {
		critter1, idx1 = cs.Select(rng, cfg)
		// critter2, idx2 = cs.Select(rng, cfg)
		critter2, idx2 = cs.SelectFar(rng, cfg, critter1)
	}
	off1, off2 := critter1.Cross(rng, critter2, cfg)

	parent1, parent2 := cs[idx1], cs[idx2]

	_, killIdx1 := cs.SelectRandom(rng)
	_, killIdx2 := cs.SelectRandom(rng)
	cs[killIdx1], cs[killIdx2] = off1, off2

	return Critters{off1, off2}, Critters{parent1, parent2}
}

type Stats struct {
	AvgErr                 ewma.MovingAverage
	AvgStepsPerInp         ewma.MovingAverage
	BetterThanParentsRatio ewma.MovingAverage
	StepTime               ewma.MovingAverage
	AvgNLowErr             ewma.MovingAverage
	BestError              ewma.MovingAverage
	Custom                 map[string]ewma.MovingAverage
	LowErr                 Critters
	Generation             uint32
	cfg                    *config.Options
}

func (st *Stats) customErrString() string {
	custErr := make([]string, 0, len(st.Custom))
	for k, v := range st.Custom {
		custErr = append(custErr, k+"="+strconv.FormatFloat(v.Value(), 'f', 3, 64))
	}
	sort.Strings(custErr)
	return strings.Join(custErr, " ")
}

func (st *Stats) String() string {
	avgNLowErr := st.AvgNLowErr.Value()
	return fmt.Sprintf("gen %4d - avgErr %1.3f (avg best %1.3f) - err<%1.2f = %.2f pct (%2d) - offspringsBetter %2.2f pct \navgNSteps/inp %2.1f - step time %s / critter - mutSigma %2.5f", st.Generation, st.AvgErr.Value(), st.BestError.Value(), st.cfg.ErrThreshold, (avgNLowErr/float64(st.cfg.PopSize))*100, int(avgNLowErr), st.BetterThanParentsRatio.Value()*100, st.AvgStepsPerInp.Value(), time.Duration(st.StepTime.Value()), st.cfg.MutSigmaRatio)
}

func (st *Stats) CustomStat(name string, age ...float64) ewma.MovingAverage {
	if st.Custom == nil {
		st.Custom = make(map[string]ewma.MovingAverage)
	}
	c := st.Custom
	ma, ok := c[name]
	if !ok {
		ma = ewma.NewMovingAverage(age...)
		c[name] = ma
	}
	return ma
}

func timer() func(perN int) time.Duration {
	start := time.Now()
	return func(perN int) time.Duration {
		return time.Duration(int64(time.Now().Sub(start)) / int64(perN))
	}
}

func NewRNG(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(maybeUnixNano(seed)))
}

func OpGenerator(rng *rand.Rand) func() vm.Op {
	return func() vm.Op {
		return vm.Ops[opNames[rng.Intn(len(opNames))]]
	}
}

func RandCritters(cfg *config.Options, rng *rand.Rand) Critters {
	cg := CritterGenerator(cfg, rng)
	p := NewCritters(cfg.PopSize)
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

func SortErrorGen(seed int64, cfg *config.Options) ErrorFunction {
	fatalErrs := cfg.FatalErrors
	minLen := cfg.MinTrainArrLen
	sizeRange := 1 + (cfg.MaxTrainArrLen - minLen)
	slGen := genTestSlice
	if minLen == cfg.MaxTrainArrLen {
		slGen = bufferedSliceGen(minLen, seed)
	}
	return func(c *Critter, input ...int) float64 {
		rng := rand.New(rand.NewSource(maybeUnixNano(seed)))
		inpLen := rng.Intn(sizeRange) + minLen
		var inp, want []int
		if len(input) == 0 {
			inp, want = slGen(inpLen, rng)
		} else {
			inp = input
			want = make([]int, len(input))
			copy(want, input)
			sort.Ints(want)
		}

		_, err := c.Input(inp, vm.StackInt).Run(fatalErrs)
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
		// return levenshtein(want, outp)
	}
}

// bufferedSliceGen pregenerates slices of inpLen
func bufferedSliceGen(inpLen int, seed int64) func(_ int, _ *rand.Rand) (inp []int, want []int) {
	type holder [2][]int
	slices := make(chan holder, 1024)
	go func(ch chan<- holder) {
		rng := rand.New(rand.NewSource(maybeUnixNano(seed)))
		for {
			in, want := genTestSlice(inpLen, rng)
			ch <- [2][]int{in, want}
		}
	}(slices)

	return func(_ int, _ *rand.Rand) (inp []int, want []int) {
		inWant := <-slices
		return inWant[0], inWant[1]
	}
}

func genTestSlice(inpLen int, rng *rand.Rand) (inp []int, want []int) {
	if inpLen == 0 {
		panic("wwadas")
	}
	inp = make([]int, inpLen)
	for i := range inp {
		inp[i] = rng.Intn(21)
	}
	want = make([]int, inpLen)
	copy(want, inp)
	sort.Ints(want)
	if positionalError(inp, want) < 0.8 {
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
	if len == 0 {
		return 0
	}
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
		return 1
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

	// swap to save some memory O(min(a,b)) instead of O(a)
	if len(s1) > len(s2) {
		s1, s2 = s2, s1
		lenS1, lenS2 = lenS2, lenS1
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
