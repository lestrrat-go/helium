package helium

import (
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

// The cycle guard answers an ancestor question from the operand's own claim
// slots, so every way a node can name another as its parent has to be covered
// by the same downward enumeration. Hand-written cases only cover the shapes
// somebody thought of, and each shape that was missed looked exactly like the
// ones that were not. This file drives the guard against its unconditional
// form over randomly built trees instead, so a slot the enumeration forgets is
// found by a test run rather than by a reader.

// cycleRandomOps is how many mutation rounds each seed drives.
const cycleRandomOps = 120

// cycleRandomPairsPerOp is how many (insertion point, operand) pairs each round
// compares the two guards on. The comparison is independent of whether the
// mutation that follows would succeed, so sampling several pairs per round
// covers far more configurations than one pair per mutation would.
const cycleRandomPairsPerOp = 4

// cycleRandomPoolCap bounds the live node pool. It is deliberately SMALL: the
// interesting configurations are the ones where a randomly drawn pair happens
// to sit on the same chain, and a small pool makes that far likelier per draw
// than a large one does.
const cycleRandomPoolCap = 28

// cycleRandomDefaultSeeds is the seed count the ordinary test run uses. It runs
// in well under a second, and it is roughly ten times the largest seed any of
// this guard's known past defects needed: reintroducing them one at a time into
// the current code, the latest first failure was at seed 371.
const cycleRandomDefaultSeeds = 4000

// cycleRandomShortSeeds is the seed count under -short.
const cycleRandomShortSeeds = 200

// cycleRandomSeedsEnv names the environment variable that overrides the seed
// count, for a longer soak than the normal suite can afford.
const cycleRandomSeedsEnv = "HELIUM_CYCLE_RANDOM_SEEDS"

// cycleRandomParentCap bounds the parent walk the driver uses to prove a
// mutation left no live parent loop behind. The pool never holds more than
// cycleRandomPoolCap nodes, so no legitimate chain can be longer.
const cycleRandomParentCap = 4 * cycleRandomPoolCap

// cycleRandomSeedCount reports how many seeds this run drives. Seeds are the
// consecutive integers below it, so a run is fully reproducible from the count
// alone and a failure names the one seed to replay.
func cycleRandomSeedCount(t *testing.T) int {
	t.Helper()

	if v := os.Getenv(cycleRandomSeedsEnv); v != "" {
		n, err := strconv.Atoi(v)
		require.NoError(t, err, "%s must be an integer seed count", cycleRandomSeedsEnv)
		require.Positive(t, n, "%s must be positive", cycleRandomSeedsEnv)
		return n
	}
	if testing.Short() {
		return cycleRandomShortSeeds
	}
	return cycleRandomDefaultSeeds
}

// cycleWorld is one seed's randomly built world: a few documents, a pool of
// nodes drawn from them, and the log of what was done to get there.
type cycleWorld struct {
	t    *testing.T
	rng  *rand.Rand
	seed uint64
	docs []*Document
	pool []Node
	log  []string
	next int
}

// newCycleWorld seeds a world with the shapes an ordinary parse produces —
// an internal subset, an entity, and an entity reference expanded inside an
// attribute value — because those are the child links that do NOT name their
// holder back, and the guard has to tell them from the ones that do.
func newCycleWorld(t *testing.T, seed uint64) *cycleWorld {
	t.Helper()

	w := &cycleWorld{
		t:    t,
		rng:  rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)),
		seed: seed,
	}

	parsed, err := NewParser().Parse(t.Context(), []byte(
		`<!DOCTYPE root [<!ENTITY e "val">]><root x="&e;"><child a="&e;"/></root>`))
	require.NoError(t, err)

	plain := NewDocument("1.0", "UTF-8", StandaloneImplicitNo)
	w.docs = []*Document{parsed, plain}

	for _, doc := range w.docs {
		w.push(doc)
	}
	if ent, ok := parsed.GetEntity("e"); ok {
		// The parser records an entity referenced from an attribute value with a
		// firstChild and NO lastChild (Document.stringToNodeList). That is the
		// shape an append turns into an off-chain parent claim, so seed the
		// expansion as well as the entity.
		w.push(ent)
		for c := range Children(ent) {
			w.push(c)
		}
	}
	if root := parsed.DocumentElement(); root != nil {
		w.push(root)
		for c := range Children(root) {
			w.push(c)
		}
	}
	if dtd := parsed.IntSubset(); dtd != nil {
		w.push(dtd)
	}
	return w
}

// push adds n to the pool, evicting a random entry once the pool is full so the
// pool keeps churning instead of freezing into its first few nodes.
func (w *cycleWorld) push(n Node) {
	if isNilNode(n) {
		return
	}
	if len(w.pool) < cycleRandomPoolCap {
		w.pool = append(w.pool, n)
		return
	}
	w.pool[w.rng.IntN(len(w.pool))] = n
}

// pick returns a random pool node.
func (w *cycleWorld) pick() Node {
	return w.pool[w.rng.IntN(len(w.pool))]
}

// pickMutable returns a random pool node that can be mutated, or nil when the
// draw did not land on one. A nil result skips the round; retrying until it
// lands would bias the draw toward whatever the pool holds most of.
func (w *cycleWorld) pickMutable() MutableNode {
	mn, _ := w.pick().(MutableNode)
	return mn
}

// pickOperand returns a random pool node that may be LINKED somewhere, or nil
// when the draw missed. A *Document is excluded: nothing in this package links
// a document node into another tree, and doing so makes setTreeDoc recurse
// through the document's own children forever, which is a hazard of that
// function and not of the guard this file tests.
func (w *cycleWorld) pickOperand() Node {
	n := w.pick()
	if _, ok := n.(*Document); ok {
		return nil
	}
	return n
}

// pickElement returns a random pool element, or nil when the draw missed.
func (w *cycleWorld) pickElement() *Element {
	e, _ := w.pick().(*Element)
	return e
}

// pickDoc returns one of the world's documents.
func (w *cycleWorld) pickDoc() *Document {
	return w.docs[w.rng.IntN(len(w.docs))]
}

// notef records one step so a failure can say how the tree was reached.
func (w *cycleWorld) notef(format string, args ...any) {
	w.log = append(w.log, fmt.Sprintf(format, args...))
}

// describe names a node for a failure message.
func describeCycleNode(n Node) string {
	if isNilNode(n) {
		return "<nil>"
	}
	return fmt.Sprintf("%s(%s)@%p", n.Type(), n.Name(), n.baseDocNode())
}

// compareGuards is the assertion this whole file exists for: on a randomly
// drawn (insertion point, operand) pair the shortcut guard must reach the same
// verdict as the unconditional walk, whatever the tree looks like.
func (w *cycleWorld) compareGuards() {
	var parent Node
	// A nil insertion point is a real input to the guard, so draw it sometimes.
	if w.rng.IntN(16) != 0 {
		parent = w.pick()
	}
	cur := w.pick()

	want := referenceWouldCreateCycle(parent, cur)
	got := wouldCreateCycle(parent, cur)
	if want == got {
		return
	}
	w.t.Fatalf("seed %d: wouldCreateCycle(%s, %s) = %v, unconditional walk = %v\nhistory:\n  %s",
		w.seed, describeCycleNode(parent), describeCycleNode(cur), got, want,
		strings.Join(w.log, "\n  "))
}

// requireNoParentLoop proves the mutation just performed did not leave a live
// parent-pointer loop behind. A loop would also hang the unconditional walk on
// the next round, so this check both reports the corruption and stops the run
// from wedging on it.
func (w *cycleWorld) requireNoParentLoop(n Node) {
	if isNilNode(n) {
		return
	}
	steps := 0
	for anc := n; anc != nil; anc = anc.Parent() {
		steps++
		if steps <= cycleRandomParentCap {
			continue
		}
		w.t.Fatalf("seed %d: parent walk from %s did not terminate in %d steps\nhistory:\n  %s",
			w.seed, describeCycleNode(n), cycleRandomParentCap, strings.Join(w.log, "\n  "))
	}
}

// mutate performs one random mutation. Errors are expected and ignored: most of
// the value is in the operations the API REFUSES, because a refusal is the
// guard firing and the next round checks the tree it left behind.
func (w *cycleWorld) mutate() {
	switch w.rng.IntN(12) {
	case 0:
		w.opCreateElement()
	case 1:
		w.opCreateLeaf()
	case 2, 3:
		w.opSetAttribute()
	case 4, 5:
		w.opAddChild()
	case 6:
		w.opAddSibling()
	case 7:
		w.opReplace()
	case 8:
		w.opUnlink()
	case 9:
		w.opMoveDocument()
	case 10:
		w.opSubsetOrReference()
	case 11:
		w.opEmptyChildList()
	}
}

func (w *cycleWorld) opCreateElement() {
	doc := w.pickDoc()
	w.next++
	e, err := doc.CreateElement(fmt.Sprintf("e%d", w.next))
	if err != nil {
		return
	}
	w.notef("create element e%d", w.next)
	w.push(e)
}

func (w *cycleWorld) opCreateLeaf() {
	doc := w.pickDoc()
	w.next++
	switch w.rng.IntN(3) {
	case 0:
		w.notef("create text t%d", w.next)
		w.push(doc.CreateText(fmt.Appendf(nil, "t%d", w.next)))
	case 1:
		w.notef("create comment c%d", w.next)
		w.push(doc.CreateComment(fmt.Appendf(nil, "c%d", w.next)))
	default:
		w.notef("create pi p%d", w.next)
		w.push(doc.CreatePI(fmt.Sprintf("p%d", w.next), "d"))
	}
}

func (w *cycleWorld) opSetAttribute() {
	e := w.pickElement()
	if e == nil {
		return
	}
	w.next++
	name := fmt.Sprintf("a%d", w.next)
	if err := e.SetAttribute(name, "v"); err != nil {
		return
	}
	w.notef("%s.SetAttribute(%s)", describeCycleNode(e), name)
	for _, attr := range e.Attributes() {
		if attr.Name() == name {
			w.push(attr)
			if c := attr.FirstChild(); c != nil {
				w.push(c)
			}
			return
		}
	}
}

func (w *cycleWorld) opAddChild() {
	parent := w.pickMutable()
	cur := w.pickOperand()
	if parent == nil || isNilNode(cur) {
		return
	}
	if err := parent.AddChild(cur); err != nil {
		return
	}
	w.notef("%s.AddChild(%s)", describeCycleNode(parent), describeCycleNode(cur))
	w.requireNoParentLoop(cur)
	w.requireNoParentLoop(parent)
}

func (w *cycleWorld) opAddSibling() {
	anchor := w.pickMutable()
	cur := w.pickOperand()
	if anchor == nil || isNilNode(cur) {
		return
	}
	if err := anchor.AddSibling(cur); err != nil {
		return
	}
	w.notef("%s.AddSibling(%s)", describeCycleNode(anchor), describeCycleNode(cur))
	w.requireNoParentLoop(cur)
	w.requireNoParentLoop(anchor)
}

func (w *cycleWorld) opReplace() {
	victim := w.pickMutable()
	cur := w.pickOperand()
	if victim == nil || isNilNode(cur) {
		return
	}
	if err := victim.Replace(cur); err != nil {
		return
	}
	w.notef("%s.Replace(%s)", describeCycleNode(victim), describeCycleNode(cur))
	w.requireNoParentLoop(cur)
	w.requireNoParentLoop(victim)
}

func (w *cycleWorld) opUnlink() {
	n := w.pickMutable()
	if n == nil {
		return
	}
	if _, ok := n.(*Document); ok {
		return
	}
	w.notef("UnlinkNode(%s)", describeCycleNode(n))
	UnlinkNode(n)
}

// opMoveDocument moves a node between documents, or out of every document. The
// off-chain-claim record the guard bails out on lives on the owning document,
// so a move is the one operation that can separate a claim from the flag the
// guard reads for it.
//
// It moves the node ALONE, through SetOwnerDocument, rather than its subtree
// through SetTreeDoc. Both carry the record the same way (adoptOffChainClaims),
// and both put the destination document in the same state, but SetTreeDoc
// recurses through the child graph with no cross-level visited set, so a shared
// entity holding a reference back to itself — a shape these random trees do
// build, and one the cycle guard is right to allow because it closes no parent
// loop — sends it into unbounded recursion. That is a property of SetTreeDoc,
// not of the guard, so this driver does not go through it.
func (w *cycleWorld) opMoveDocument() {
	n := w.pickMutable()
	if n == nil {
		return
	}
	if _, ok := n.(*Document); ok {
		return
	}
	var doc *Document
	if w.rng.IntN(2) != 0 {
		doc = w.pickDoc()
	}
	w.notef("%s.SetOwnerDocument(%p)", describeCycleNode(n), doc)
	n.SetOwnerDocument(doc)
}

// opEmptyChildList unlinks every child of a node. An empty child list is the
// precondition for the shortcut the guard takes, so driving nodes INTO that
// state is what puts the shortcut under test; unlinking one random pool node at
// a time rarely empties anything.
func (w *cycleWorld) opEmptyChildList() {
	n := w.pickMutable()
	if n == nil {
		return
	}
	var kids []Node
	for c := range Children(n) {
		kids = append(kids, c)
	}
	if len(kids) == 0 {
		return
	}
	w.notef("empty child list of %s", describeCycleNode(n))
	for _, c := range kids {
		mc, ok := c.(MutableNode)
		if !ok {
			continue
		}
		UnlinkNode(mc)
		w.push(c)
	}
}

// opSubsetOrReference builds the shapes only DTDs and entities produce: a
// subset installed on a document from a slot rather than a child list, a copied
// external subset (which names the destination document while staying off its
// child list), a declaration registered into a DTD, and an entity reference
// whose child is a shared entity the DTD owns.
func (w *cycleWorld) opSubsetOrReference() {
	doc := w.pickDoc()
	switch w.rng.IntN(4) {
	case 0:
		dtd, err := doc.CreateInternalSubset("root", "", "")
		if err != nil {
			return
		}
		w.notef("CreateInternalSubset on %p", doc)
		w.push(dtd)
	case 1:
		src := NewDefaultDocument()
		srcDTD := newDTD()
		srcDTD.name = "root"
		src.extSubset = srcDTD
		CopyExtSubset(src, doc)
		w.notef("CopyExtSubset into %p", doc)
		w.push(doc.ExtSubset())
	case 2:
		dtd := doc.IntSubset()
		if dtd == nil {
			return
		}
		w.next++
		name := fmt.Sprintf("ent%d", w.next)
		ent, err := dtd.AddEntity(name, enum.InternalGeneralEntity, "", "", "v")
		if err != nil {
			return
		}
		w.notef("AddEntity(%s)", name)
		w.push(ent)
	default:
		dtd := doc.IntSubset()
		if dtd == nil {
			return
		}
		var name string
		for e := range Children(dtd) {
			if e.Type() == EntityDeclNode {
				name = e.Name()
				break
			}
		}
		if name == "" {
			return
		}
		ref, err := doc.CreateReference(name)
		if err != nil {
			return
		}
		w.notef("CreateReference(%s)", name)
		w.push(ref)
	}
}

// TestWouldCreateCycleMatchesUnconditionalWalkRandomized is the differential
// driver for the cycle guard. It builds random trees through the exported
// mutation API and, on randomly drawn (insertion point, operand) pairs, asserts
// that wouldCreateCycle answers exactly what referenceWouldCreateCycle answers.
//
// It is deterministic: the seeds are the consecutive integers below the seed
// count, every draw comes from a PCG source keyed on the seed, and nothing
// consults the clock, the map iteration order, or the goroutine schedule. A
// failure names the seed and prints the mutation history that reached the tree,
// so it replays exactly. Set HELIUM_CYCLE_RANDOM_SEEDS to soak it longer than
// the ordinary suite can afford.
//
// It does NOT run in parallel, because it resets unownedOffChainClaim: that
// flag is package-level, sticky, and never cleared in production, so a claim
// made by any earlier test would otherwise send every operand down the walk and
// silently stop this test exercising the shortcut at all.
func TestWouldCreateCycleMatchesUnconditionalWalkRandomized(t *testing.T) {
	restore := unownedOffChainClaim.Load()
	t.Cleanup(func() { unownedOffChainClaim.Store(restore) })

	seeds := cycleRandomSeedCount(t)
	for seed := range uint64(seeds) {
		// Each seed starts from a clean flag so its verdicts depend only on the
		// tree it built, never on what a previous seed happened to do.
		unownedOffChainClaim.Store(false)

		w := newCycleWorld(t, seed)
		for range cycleRandomOps {
			for range cycleRandomPairsPerOp {
				w.compareGuards()
			}
			w.mutate()
		}
	}
}
