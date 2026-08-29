package helium

import (
	"fmt"
	"testing"

	"github.com/lestrrat-go/helium/enum"
	"github.com/stretchr/testify/require"
)

// TestDTDReplacementGraphValidationVisitsAChainOnce guards the shared session
// state used across declaration roots. Each entity after the first refers to
// its predecessor, so restarting the traversal for every root would visit a
// triangular number of nodes. Shared done state visits each declaration and
// each replacement root exactly once.
func TestDTDReplacementGraphValidationVisitsAChainOnce(t *testing.T) {
	const entityCount = 512

	doc := NewDefaultDocument()
	dtd, err := doc.CreateInternalSubset("root", "", "")
	require.NoError(t, err)

	entities := make([]*Entity, 0, entityCount)
	for i := range entityCount {
		entity, addErr := dtd.AddEntity(
			fmt.Sprintf("e%d", i), enum.InternalGeneralEntity, "", "", "",
		)
		require.NoError(t, addErr)
		entities = append(entities, entity)
	}

	replacements := make([]dtdReplacementCopy, 0, entityCount)
	for i, entity := range entities {
		var child Node = doc.CreateText([]byte("base"))
		if i > 0 {
			child, err = doc.CreateReference(entities[i-1].Name())
			require.NoError(t, err)
		}
		replacements = append(replacements, dtdReplacementCopy{
			entity:   entity,
			children: []Node{child},
		})
	}

	validator := newDTDReplacementGraphValidator(replacements)
	require.False(t, validator.hasCycle())
	require.Equal(t, entityCount*2, validator.visits,
		"each entity and replacement root must be visited once across the full chain")
}
