package plugin

import (
	"fmt"
	"strconv"
	"strings"
)

type semver struct {
	major int
	minor int
	patch int
}

func (v semver) compare(other semver) int {
	if v.major != other.major {
		if v.major < other.major {
			return -1
		}
		return 1
	}
	if v.minor != other.minor {
		if v.minor < other.minor {
			return -1
		}
		return 1
	}
	if v.patch != other.patch {
		if v.patch < other.patch {
			return -1
		}
		return 1
	}
	return 0
}

// ValidateCompatibility checks whether compatibility_skoll matches the provided core version.
// A blank compatibility or blank core version is treated as compatible.
func (i Info) ValidateCompatibility(coreVersion string) error {
	raw := strings.TrimSpace(i.CompatibilitySkoll)
	if raw == "" {
		return nil
	}
	if strings.TrimSpace(coreVersion) == "" {
		return nil
	}

	core, err := parseSemver(coreVersion)
	if err != nil {
		return fmt.Errorf("invalid core version %q: %w", coreVersion, err)
	}
	constraints, err := parseConstraints(raw)
	if err != nil {
		return fmt.Errorf("invalid compatibility_skoll %q: %w", raw, err)
	}
	for _, c := range constraints {
		cmp := core.compare(c.version)
		if !matchesConstraint(c.op, cmp) {
			return fmt.Errorf("core version %s does not satisfy %s%s", normalizeSemver(coreVersion), c.op, c.rawVersion)
		}
	}
	return nil
}

type constraint struct {
	op         string
	version    semver
	rawVersion string
}

func parseConstraints(raw string) ([]constraint, error) {
	tokens := strings.Fields(strings.TrimSpace(raw))
	if len(tokens) == 0 {
		return nil, fmt.Errorf("empty compatibility expression")
	}

	out := make([]constraint, 0, len(tokens))
	for _, token := range tokens {
		t := strings.TrimSpace(token)
		if t == "" {
			continue
		}

		op := "="
		switch {
		case strings.HasPrefix(t, ">="):
			op = ">="
			t = strings.TrimPrefix(t, ">=")
		case strings.HasPrefix(t, "<="):
			op = "<="
			t = strings.TrimPrefix(t, "<=")
		case strings.HasPrefix(t, ">"):
			op = ">"
			t = strings.TrimPrefix(t, ">")
		case strings.HasPrefix(t, "<"):
			op = "<"
			t = strings.TrimPrefix(t, "<")
		case strings.HasPrefix(t, "="):
			op = "="
			t = strings.TrimPrefix(t, "=")
		}

		vraw := normalizeSemver(t)
		v, err := parseSemver(vraw)
		if err != nil {
			return nil, err
		}
		out = append(out, constraint{op: op, version: v, rawVersion: vraw})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("empty compatibility expression")
	}
	return out, nil
}

func matchesConstraint(op string, cmp int) bool {
	switch op {
	case ">":
		return cmp > 0
	case ">=":
		return cmp >= 0
	case "<":
		return cmp < 0
	case "<=":
		return cmp <= 0
	case "=":
		return cmp == 0
	default:
		return false
	}
}

func parseSemver(raw string) (semver, error) {
	normalized := normalizeSemver(raw)
	parts := strings.Split(normalized, ".")
	if len(parts) != 3 {
		return semver{}, fmt.Errorf("must be semver major.minor.patch")
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return semver{}, err
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return semver{}, err
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return semver{}, err
	}
	if major < 0 || minor < 0 || patch < 0 {
		return semver{}, fmt.Errorf("version segment cannot be negative")
	}
	return semver{major: major, minor: minor, patch: patch}, nil
}

func normalizeSemver(raw string) string {
	v := strings.TrimSpace(raw)
	v = strings.TrimPrefix(strings.ToLower(v), "v")
	if idx := strings.IndexAny(v, "+-"); idx >= 0 {
		v = v[:idx]
	}
	return v
}
