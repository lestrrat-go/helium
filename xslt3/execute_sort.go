package xslt3

import (
	"context"

	"github.com/lestrrat-go/helium/xpath3"
	"github.com/lestrrat-go/helium/internal/sequence"
)

func (ec *execContext) execPerformSort(ctx context.Context, inst *PerformSortInst) error {
	var seq xpath3.Sequence

	if inst.Select != nil {
		result, err := ec.evalXPath(nil, inst.Select, ec.contextNode)
		if err != nil {
			return err
		}
		seq = result.Sequence()
	} else if len(inst.Body) > 0 {
		// Body acts as sequence constructor: evaluate items individually
		// so that each text item remains a separate sortable unit.
		var err error
		seq, err = ec.evaluateBodyAsSequence(ctx, inst.Body)
		if err != nil {
			return err
		}
	}
	if seq == nil || sequence.Len(seq) == 0 {
		return nil
	}

	// Try to extract nodes for node-based sorting
	nodes, allNodes := xpath3.NodesFrom(seq)
	if allNodes && len(nodes) > 0 {
		if len(inst.Sort) > 0 {
			var err error
			nodes, err = sortNodes(ctx, ec, nodes, inst.Sort)
			if err != nil {
				return err
			}
		}

		savedCurrent := ec.currentNode
		savedContext := ec.contextNode
		savedPos := ec.position
		savedSize := ec.size
		ec.size = len(nodes)
		defer func() {
			ec.currentNode = savedCurrent
			ec.contextNode = savedContext
			ec.position = savedPos
			ec.size = savedSize
		}()

		// Output sorted nodes
		for _, node := range nodes {
			if err := ec.copyNodeToOutput(node); err != nil {
				return err
			}
		}
		return nil
	}

	// Atomic sequence: sort by string value and output as text items
	if len(inst.Sort) > 0 {
		var err error
		seq, err = sortItems(ctx, ec, seq, inst.Sort)
		if err != nil {
			return err
		}
	}

	// In capture mode (e.g. inside xsl:function), push sorted items
	// directly so the caller receives a proper sequence of atomic values
	// rather than merged text nodes.
	out := ec.currentOutput()
	if out.captureItems && out.doc != nil && out.current == out.doc.DocumentElement() {
		out.pendingItems = append(out.pendingItems, sequence.Materialize(seq)...)
		if seq != nil && sequence.Len(seq) > 0 {
			out.noteOutput()
		}
		return nil
	}

	// Output atomic items separated by spaces
	idx := 0
	for item := range sequence.Items(seq) {
		if idx > 0 {
			sep, err := ec.resultDoc.CreateText([]byte(" "))
			if err != nil {
				return err
			}
			if err := ec.addNode(sep); err != nil {
				return err
			}
		}
		av, ok := item.(xpath3.AtomicValue)
		if !ok {
			continue
		}
		s, err := xpath3.AtomicToString(av)
		if err != nil {
			continue
		}
		text, err := ec.resultDoc.CreateText([]byte(s))
		if err != nil {
			return err
		}
		if err := ec.addNode(text); err != nil {
			return err
		}
		idx++
	}
	return nil
}

