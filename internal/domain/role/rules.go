package role

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var roleKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_:.\-]{1,63}$`)

func ValidateName(name string) error {
	v := strings.TrimSpace(name)
	if v == "" {
		return fmt.Errorf("role name is required")
	}
	if len([]rune(v)) > 64 {
		return fmt.Errorf("role name is too long")
	}
	return nil
}

func ValidateKey(key string) error {
	v := strings.TrimSpace(key)
	if !roleKeyPattern.MatchString(v) {
		return fmt.Errorf("role key must match %s", roleKeyPattern.String())
	}
	return nil
}

func NormalizePermissions(raw []string) []string {
	uniq := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		v := strings.TrimSpace(strings.ToLower(p))
		if v == "" {
			continue
		}
		if _, ok := uniq[v]; ok {
			continue
		}
		uniq[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}
