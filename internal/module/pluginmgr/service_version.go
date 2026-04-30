package pluginmgr

import "strings"

func isValidVersion(version string) bool {
	v := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(version, "v"), "V"))
	if v == "" {
		return false
	}
	parts := strings.Split(v, ".")
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, ch := range p {
			if ch < '0' || ch > '9' {
				return false
			}
		}
	}
	return true
}

func compareVersion(current, latest string) int {
	c := parseVersionParts(current)
	l := parseVersionParts(latest)
	maxLen := len(c)
	if len(l) > maxLen {
		maxLen = len(l)
	}
	for i := 0; i < maxLen; i++ {
		cv := 0
		if i < len(c) {
			cv = c[i]
		}
		lv := 0
		if i < len(l) {
			lv = l[i]
		}
		if cv < lv {
			return -1
		}
		if cv > lv {
			return 1
		}
	}
	return 0
}

func parseVersionParts(version string) []int {
	v := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(version, "v"), "V"))
	parts := strings.Split(v, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		var n int
		for _, ch := range p {
			n = n*10 + int(ch-'0')
		}
		out = append(out, n)
	}
	return out
}

func verifySignature(packageHash, signature string) bool {
	packageHash = strings.TrimSpace(packageHash)
	signature = strings.TrimSpace(signature)
	if packageHash == "" || signature == "" {
		return false
	}
	return signature == "sig:"+packageHash
}
