package xpath3_test

// Shared test-only string constants that recur across the package's test files.
// Centralizing them keeps goconst happy and avoids drift between expectations.
const (
	wantTrue  = "true"
	wantFalse = "false"
	wantNaN   = "NaN"
	wantINF   = "INF"
	want1Dot5 = "1.5"
	expr1To10 = "1 to 10"
)
