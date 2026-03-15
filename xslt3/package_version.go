package xslt3

import (
	"strconv"
	"strings"
)

// PackageVersion represents a parsed XSLT 3.0 package version.
// Version format: NumericPart ("-" NamePart)?
// NumericPart: Integer ("." Integer)*
// NamePart: NCName
type PackageVersion struct {
	Numbers []int
	Name    string // name suffix after "-" (e.g., "alpha" in "1.0-alpha")
	Raw     string // original version string
}

// ParsePackageVersion parses a version string like "1.0.0" or "2.0.0-alpha".
func ParsePackageVersion(s string) PackageVersion {
	s = strings.TrimSpace(s)
	if s == "" {
		return PackageVersion{Raw: s}
	}

	pv := PackageVersion{Raw: s}

	// Split on first "-" for name suffix
	numPart := s
	if idx := strings.IndexByte(s, '-'); idx >= 0 {
		numPart = s[:idx]
		pv.Name = s[idx+1:]
	}

	// Parse numeric parts
	for _, part := range strings.Split(numPart, ".") {
		n, err := strconv.Atoi(part)
		if err != nil {
			// Non-numeric version component - treat as name
			pv.Name = s
			pv.Numbers = nil
			return pv
		}
		pv.Numbers = append(pv.Numbers, n)
	}

	return pv
}

// Compare compares two versions. Returns -1, 0, or 1.
// Missing trailing components are treated as zero.
func (v PackageVersion) Compare(other PackageVersion) int {
	maxLen := len(v.Numbers)
	if len(other.Numbers) > maxLen {
		maxLen = len(other.Numbers)
	}

	for i := 0; i < maxLen; i++ {
		a := 0
		if i < len(v.Numbers) {
			a = v.Numbers[i]
		}
		b := 0
		if i < len(other.Numbers) {
			b = other.Numbers[i]
		}
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
	}

	// Numeric parts equal; compare name suffix
	// Version with no suffix > version with suffix (1.0 > 1.0-alpha)
	if v.Name == "" && other.Name != "" {
		return 1
	}
	if v.Name != "" && other.Name == "" {
		return -1
	}
	if v.Name < other.Name {
		return -1
	}
	if v.Name > other.Name {
		return 1
	}
	return 0
}

// VersionConstraint represents a package-version constraint from xsl:use-package.
type VersionConstraint struct {
	// Exact version (nil if wildcard or range)
	Exact *PackageVersion
	// Prefix match: "1.*" means Numbers=[1], matchPrefix=true
	Prefix  []int
	IsRange bool
	// Range bounds (inclusive)
	RangeFrom PackageVersion
	RangeTo   PackageVersion
	// Wildcard: "*" matches any version
	IsWildcard bool
	Raw        string
}

// ParseVersionConstraint parses a package-version attribute value.
// Forms: "*", "1.0", "1.*", "1.0 to 2.0", "" (= "*")
func ParseVersionConstraint(s string) VersionConstraint {
	s = strings.TrimSpace(s)
	if s == "" || s == "*" {
		return VersionConstraint{IsWildcard: true, Raw: s}
	}

	// Check for range: "X to Y"
	if parts := strings.SplitN(s, " to ", 2); len(parts) == 2 {
		from := ParsePackageVersion(strings.TrimSpace(parts[0]))
		to := ParsePackageVersion(strings.TrimSpace(parts[1]))
		return VersionConstraint{IsRange: true, RangeFrom: from, RangeTo: to, Raw: s}
	}

	// Check for prefix wildcard: "1.*"
	if strings.HasSuffix(s, ".*") {
		prefix := s[:len(s)-2]
		var nums []int
		for _, p := range strings.Split(prefix, ".") {
			n, err := strconv.Atoi(p)
			if err != nil {
				break
			}
			nums = append(nums, n)
		}
		if len(nums) > 0 {
			return VersionConstraint{Prefix: nums, Raw: s}
		}
	}

	// Exact version
	pv := ParsePackageVersion(s)
	return VersionConstraint{Exact: &pv, Raw: s}
}

// Matches checks if a version satisfies this constraint.
func (c VersionConstraint) Matches(v PackageVersion) bool {
	if c.IsWildcard {
		return true
	}

	if c.IsRange {
		return v.Compare(c.RangeFrom) >= 0 && v.Compare(c.RangeTo) <= 0
	}

	if c.Prefix != nil {
		// Check that the version starts with the prefix numbers
		for i, n := range c.Prefix {
			if i >= len(v.Numbers) {
				return false
			}
			if v.Numbers[i] != n {
				return false
			}
		}
		return true
	}

	if c.Exact != nil {
		return v.Compare(*c.Exact) == 0
	}

	return false
}
