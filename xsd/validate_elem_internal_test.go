package xsd

import "testing"

// TestSubstTypeDerivationBlockedNilSafe verifies the substitution-group block
// gate is nil-safe: a nil derived or base type must not panic dereferencing the
// BaseType chain. It returns false (not blocked) — a missing type cannot be
// shown to be blocked.
func TestSubstTypeDerivationBlockedNilSafe(t *testing.T) {
	t.Parallel()

	nonNil := &TypeDef{}
	cases := []struct {
		name          string
		derived, base *TypeDef
	}{
		{"both nil", nil, nil},
		{"nil derived", nil, nonNil},
		{"nil base", nonNil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := substTypeDerivationBlocked(tc.derived, tc.base, 0); got {
				t.Fatalf("substTypeDerivationBlocked(%v, %v) = true, want false", tc.derived, tc.base)
			}
		})
	}
}
