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
	if np, name, ok := strings.Cut(s, "-"); ok {
		numPart = np
		pv.Name = name
	}

	// Parse numeric parts
	for part := range strings.SplitSeq(numPart, ".") {
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
	maxLen := max(len(v.Numbers), len(other.Numbers))

	for i := range maxLen {
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
	// MinVersion: "1.5+" means >= 1.5
	IsMinVersion bool
	MinVersion   PackageVersion
	// MaxOnly: "to 1.5" means <= 1.5 (no lower bound)
	IsMaxOnly  bool
	MaxVersion PackageVersion
	// Alternatives: "1.0.0, 2.0" means match any of these
	Alternatives []VersionConstraint
	// Wildcard: "*" matches any version
	IsWildcard bool
	Raw        string
}

// ParseVersionConstraint parses a package-version attribute value.
// Forms: "*", "1.0", "1.*", "1.0 to 2.0", "to 2.0", "1.5+", "1.0, 2.0", "" (= "*")
func ParseVersionConstraint(s string) VersionConstraint {
	s = strings.TrimSpace(s)
	if s == "" || s == "*" {
		return VersionConstraint{IsWildcard: true, Raw: s}
	}

	// Check for alternatives: "X, Y, ..."
	if strings.Contains(s, ",") {
		parts := strings.Split(s, ",")
		var alts []VersionConstraint
		for _, p := range parts {
			alts = append(alts, ParseVersionConstraint(strings.TrimSpace(p)))
		}
		return VersionConstraint{Alternatives: alts, Raw: s}
	}

	// Check for "to Y" (max-only, no lower bound)
	if strings.HasPrefix(s, "to ") {
		to := ParsePackageVersion(strings.TrimSpace(s[3:]))
		return VersionConstraint{IsMaxOnly: true, MaxVersion: to, Raw: s}
	}

	// Check for range: "X to Y"
	if parts := strings.SplitN(s, " to ", 2); len(parts) == 2 {
		from := ParsePackageVersion(strings.TrimSpace(parts[0]))
		to := ParsePackageVersion(strings.TrimSpace(parts[1]))
		return VersionConstraint{IsRange: true, RangeFrom: from, RangeTo: to, Raw: s}
	}

	// Check for minimum version: "X+"
	if strings.HasSuffix(s, "+") {
		minVer := ParsePackageVersion(s[:len(s)-1])
		return VersionConstraint{IsMinVersion: true, MinVersion: minVer, Raw: s}
	}

	// Check for prefix wildcard: "1.*"
	if strings.HasSuffix(s, ".*") {
		prefix := s[:len(s)-2]
		var nums []int
		for p := range strings.SplitSeq(prefix, ".") {
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

	if len(c.Alternatives) > 0 {
		for _, alt := range c.Alternatives {
			if alt.Matches(v) {
				return true
			}
		}
		return false
	}

	if c.IsRange {
		return v.Compare(c.RangeFrom) >= 0 && v.Compare(c.RangeTo) <= 0
	}

	if c.IsMinVersion {
		return v.Compare(c.MinVersion) >= 0
	}

	if c.IsMaxOnly {
		return v.Compare(c.MaxVersion) <= 0
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
