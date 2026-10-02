package xpath1

// CountOpsForTesting charges n operations, against limit, to an op counter that
// already stands at count. It returns the counter afterwards and the charge's
// error, so a test can start the counter next to math.MaxInt without running
// that many operations.
func CountOpsForTesting(count, n, limit int) (int, error) {
	ec := &evalContext{opCount: &count, opLimit: limit}
	err := ec.countOps(n)
	return count, err
}
