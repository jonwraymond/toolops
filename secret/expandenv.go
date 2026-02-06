package secret

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

// ExpandEnvStrict expands environment variables in s.
//
// Semantics:
//   - `$VAR` and `${VAR}` are expanded via os.ExpandEnv.
//   - If `${VAR}` is present but VAR is missing from the environment, it errors.
//   - `$$` emits a literal `$` (escape hatch).
func ExpandEnvStrict(s string) (string, error) {
	const dollarSentinel = "\x00TOOLOPS_SECRET_DOLLAR\x00"
	s = strings.ReplaceAll(s, "$$", dollarSentinel)

	missing := make(map[string]struct{})
	for _, match := range envVarPattern.FindAllStringSubmatch(s, -1) {
		key := match[1]
		if _, ok := os.LookupEnv(key); !ok {
			missing[key] = struct{}{}
		}
	}
	if len(missing) > 0 {
		keys := make([]string, 0, len(missing))
		for k := range missing {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return "", fmt.Errorf("missing required environment variables: %s", strings.Join(keys, ", "))
	}

	s = os.ExpandEnv(s)
	s = strings.ReplaceAll(s, dollarSentinel, "$")
	return s, nil
}
