// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

import (
	"encoding/json"
	"strings"
)

// responseEnvelope is the structured output the prompt asks the agent to emit.
type responseEnvelope struct {
	MQL         string `json:"mql"`
	Explanation string `json:"explanation"`
}

// parseResponse extracts the generated MQL from raw agent output. Agents are
// instructed to return a single fenced ```json block, but real-world output is
// noisy (preamble, trailing chatter, multiple fences), so parsing is defensive:
//
//  1. Prefer a fenced ```json block that decodes to the envelope.
//  2. Fall back to the last balanced {...} object anywhere in the text.
//  3. As a last resort, treat a fenced ```mql (or bare ```) block as the query.
//
// It never errors; an empty MQL in the result signals "could not parse", which
// the caller surfaces.
func parseResponse(raw string) GenResult {
	raw = strings.TrimSpace(raw)

	// 1. fenced json blocks. Agents sometimes echo an example envelope before
	// their real answer, so the LAST block with a non-empty mql wins. An
	// envelope with an empty mql is ignored so it can't shadow a later answer.
	if res, ok := lastEnvelopeWithMQL(fencedBlocks(raw, "json")); ok {
		return res
	}

	// 2. any balanced object that decodes to the envelope, last non-empty wins
	if res, ok := lastEnvelopeWithMQL(jsonObjects(raw)); ok {
		return res
	}

	// 3. fenced mql / plain code block. The bare-fence ("") case matches any
	// fence, so skip anything that is actually a JSON object — otherwise a
	// ```json block with an empty mql would be returned verbatim as the query.
	for _, lang := range []string{"mql", "coffee", ""} {
		blocks := fencedBlocks(raw, lang)
		for i := len(blocks) - 1; i >= 0; i-- {
			b := strings.TrimSpace(blocks[i])
			if b == "" || strings.HasPrefix(b, "{") {
				continue
			}
			if _, ok := decodeEnvelope(b); ok {
				continue
			}
			return GenResult{MQL: b}
		}
	}

	return GenResult{}
}

// lastEnvelopeWithMQL scans candidate strings, decoding each as the response
// envelope, and returns the last one that carries a non-empty mql. Requiring a
// non-empty mql means an explanation-only or empty envelope never shadows a
// later, real answer.
func lastEnvelopeWithMQL(candidates []string) (GenResult, bool) {
	var res GenResult
	found := false
	for _, c := range candidates {
		if env, ok := decodeEnvelope(c); ok && strings.TrimSpace(env.MQL) != "" {
			res = GenResult{
				MQL:         strings.TrimSpace(env.MQL),
				Explanation: strings.TrimSpace(env.Explanation),
			}
			found = true
		}
	}
	return res, found
}

func decodeEnvelope(s string) (responseEnvelope, bool) {
	var env responseEnvelope
	dec := json.NewDecoder(strings.NewReader(s))
	if err := dec.Decode(&env); err != nil {
		return responseEnvelope{}, false
	}
	return env, true
}

// fencedBlocks returns the contents of all ```<lang> ... ``` blocks. An empty
// lang matches a bare ``` fence (any or no language tag).
func fencedBlocks(s, lang string) []string {
	var out []string
	lines := strings.Split(s, "\n")
	inBlock := false
	var buf []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inBlock {
			if strings.HasPrefix(trimmed, "```") {
				tag := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "```")))
				if lang == "" || tag == lang {
					inBlock = true
					buf = nil
				}
			}
			continue
		}
		if strings.HasPrefix(trimmed, "```") {
			out = append(out, strings.Join(buf, "\n"))
			inBlock = false
			continue
		}
		buf = append(buf, line)
	}
	return out
}

// jsonObjects returns every top-level balanced {...} substring, in order. It is
// brace-counting with string-literal awareness so braces inside JSON strings do
// not throw off the balance.
func jsonObjects(s string) []string {
	var out []string
	depth := 0
	start := -1
	inStr := false
	escaped := false
	for i, r := range s {
		if inStr {
			switch {
			case escaped:
				escaped = false
			case r == '\\':
				escaped = true
			case r == '"':
				inStr = false
			}
			continue
		}
		switch r {
		case '"':
			inStr = true
		case '{':
			if depth == 0 {
				start = i
			}
			depth++
		case '}':
			if depth > 0 {
				depth--
				if depth == 0 && start >= 0 {
					out = append(out, s[start:i+1])
					start = -1
				}
			}
		}
	}
	return out
}
