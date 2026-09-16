package handlers

import (
	"strings"

	"github.com/gokuai/yunku-cli/internal/pipeline"
	"github.com/gokuai/yunku-cli/pkg/cmdutil"
)

// ParamNameHandler performs fuzzy correction on flag names that are
// not recognised after alias normalisation. It uses Levenshtein edit
// distance to find the closest known flag name, with a conservative
// threshold to avoid false positives.
//
// Correction rules:
//   - Edit distance must be ≤ maxEditDistance (default 2).
//   - The match must be unambiguous (exactly one candidate within threshold).
//   - Very short flag names (≤ 3 chars) use a tighter threshold of 1.
//
// This handler should run after AliasHandler in the PreParse phase
// so that obvious normalisation (camelCase → kebab-case) is already
// done and fuzzy matching only handles genuine near-misses.
type ParamNameHandler struct{}

const maxEditDistance = 2

// reservedFlags contains Cobra's built-in pseudo-flags that are registered
// lazily (inside ExecuteC) and thus absent from FlagSpecs at PreParse time.
// They must never be fuzzy-rewritten to user-defined flags.
var reservedFlags = map[string]bool{
	"help":    true,
	"version": true,
}

func (ParamNameHandler) Name() string          { return "paramname" }
func (ParamNameHandler) Phase() pipeline.Phase { return pipeline.PreParse }

func (ParamNameHandler) Handle(ctx *pipeline.Context) error {
	if len(ctx.Args) == 0 || len(ctx.FlagSpecs) == 0 {
		return nil
	}

	known := buildFlagSet(ctx.FlagSpecs)
	names := make([]string, 0, len(ctx.FlagSpecs))
	for _, spec := range ctx.FlagSpecs {
		if spec.Name != "" {
			names = append(names, spec.Name)
		}
	}

	result := make([]string, 0, len(ctx.Args))
	for _, arg := range ctx.Args {
		rewritten, ok := tryFuzzyMatch(arg, known, names)
		if ok {
			ctx.AddCorrection("paramname", pipeline.PreParse, rewritten, arg, rewritten, "fuzzy")
			result = append(result, rewritten)
		} else {
			result = append(result, arg)
		}
	}

	ctx.Args = result
	return nil
}

// tryFuzzyMatch attempts to correct an unrecognised "--flag" token by
// finding the closest known flag name within the edit distance threshold.
func tryFuzzyMatch(arg string, known map[string]bool, candidates []string) (string, bool) {
	if !strings.HasPrefix(arg, "--") {
		return "", false
	}

	bare := arg[2:]
	if bare == "" {
		return "", false
	}

	// Handle --flag=value syntax.
	var suffix string
	if idx := strings.IndexByte(bare, '='); idx >= 0 {
		suffix = bare[idx:]
		bare = bare[:idx]
	}

	// Already known — nothing to fix.
	if known[bare] {
		return "", false
	}

	// Cobra built-in flags (help, version) are not in FlagSpecs at
	// PreParse time but must never be rewritten.
	if reservedFlags[bare] {
		return "", false
	}

	threshold := maxEditDistance
	if len(bare) <= 3 {
		threshold = 1
	}

	bestDist := threshold + 1
	bestMatch := ""
	ambiguous := false

	for _, candidate := range candidates {
		dist := cmdutil.LevenshteinDist(bare, candidate)
		if dist < bestDist {
			bestDist = dist
			bestMatch = candidate
			ambiguous = false
		} else if dist == bestDist && candidate != bestMatch {
			ambiguous = true
		}
	}

	if bestDist > threshold || ambiguous || bestMatch == "" {
		return "", false
	}

	return "--" + bestMatch + suffix, true
}
