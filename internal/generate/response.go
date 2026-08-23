// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

import (
	"encoding/json"
	"regexp"
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
// Every candidate has to look like MQL before it is accepted (see plausibleMQL).
// Without that, the last-block-wins rule adopts whatever the agent printed last:
// the prompt tells it to verify with `cnspec run ... --ast` and shows the answer
// shape as `{"mql": "<the query>"}`, so an agent that shows its work ends with a
// shell transcript or the prompt's own placeholder, and that became the check.
//
// It never errors; an empty MQL in the result signals "could not parse", which
// the caller surfaces.
func parseResponse(raw string) GenResult {
	raw = strings.TrimSpace(raw)

	// 1. fenced json blocks. Agents sometimes echo an example envelope before
	// their real answer, so the LAST block with a usable mql wins. An envelope
	// whose mql is empty or implausible is ignored so it can't shadow a real
	// answer that came earlier.
	if res, ok := lastEnvelopeWithMQL(fencedBlocks(raw, "json")); ok {
		return res
	}

	// 2. any balanced object that decodes to the envelope, last usable wins
	if res, ok := lastEnvelopeWithMQL(jsonObjects(raw)); ok {
		return res
	}

	// 3. fenced mql / plain code block. The bare-fence ("") case matches any
	// fence, so skip anything that is actually a JSON object — otherwise a
	// ```json block with an empty mql would be returned verbatim as the query.
	for _, lang := range []string{"mql", "coffee", ""} {
		blocks := fencedBlocksMatching(raw, lang)
		for i := len(blocks) - 1; i >= 0; i-- {
			b := strings.TrimSpace(blocks[i].body)
			if b == "" || strings.HasPrefix(b, "{") {
				continue
			}
			if _, ok := decodeEnvelope(b); ok {
				continue
			}
			if lang == "" && nonMQLFenceTags[blocks[i].tag] {
				// a bare-fence pass matches every fence, including the ```bash
				// one holding the verification command the prompt asked for
				continue
			}
			if !plausibleMQL(b) {
				continue
			}
			return GenResult{MQL: SanitizeModelText(b)}
		}
	}

	return GenResult{}
}

// nonMQLFenceTags are language tags that cannot be MQL. The prompt asks the
// agent to run `cnspec run ... --ast`, so a ```bash or ```console block holding
// that transcript is the single most likely other fence in the response.
var nonMQLFenceTags = map[string]bool{
	"bash": true, "sh": true, "shell": true, "zsh": true, "fish": true,
	"console": true, "shell-session": true, "terminal": true, "output": true,
	"text": true, "txt": true, "log": true, "diff": true, "patch": true,
	"json": true, "yaml": true, "yml": true, "toml": true, "xml": true,
	"go": true, "python": true, "py": true, "javascript": true, "js": true,
	"typescript": true, "ts": true, "ruby": true, "rust": true, "java": true,
	"markdown": true, "md": true, "html": true, "sql": true, "powershell": true,
	"ps1": true, "dockerfile": true, "hcl": true, "terraform": true,
}

// shellTranscriptPrefixes start a line that is a command, not a query. MQL has
// no statement that begins with any of these.
var shellTranscriptPrefixes = []string{
	"$ ", "# ", "% ", "> ", "cnspec ", "cnquery ", "sudo ", "bash ", "sh ",
	"echo ", "cat ", "curl ", "npm ", "npx ", "git ", "docker ", "kubectl ",
	"export ", "cd ", "ls ", "grep ", "pip ", "brew ", "apt ", "yum ",
}

// placeholderRe matches the answer placeholders the prompt itself contains
// (`<the query>`, `<your mql>`). An agent that echoes the output format after
// its answer hands back the placeholder, and last-block-wins then stores it.
var placeholderRe = regexp.MustCompile(`(?i)<\s*(the\s+|your\s+|a\s+)?(query|mql|expression|answer)[^>]*>`)

// plausibleMQL reports whether a candidate can be a query at all. It is a
// rejection filter, not a compiler: everything it lets through still faces the
// compile gate. Its job is to stop an implausible candidate from *shadowing* a
// real answer that appeared earlier in the response.
func plausibleMQL(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if placeholderRe.MatchString(s) {
		return false
	}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		for _, prefix := range shellTranscriptPrefixes {
			if strings.HasPrefix(lower, prefix) {
				return false
			}
		}
	}
	// A query reads a field, calls something, or compares something. A bare
	// prose sentence ("I could not determine the right resource") does none of
	// those.
	return strings.ContainsAny(s, ".([") || strings.Contains(s, "==") || strings.Contains(s, "!=")
}

// lastEnvelopeWithMQL scans candidate strings, decoding each as the response
// envelope, and returns the last one that carries a usable mql. Requiring a
// plausible, non-empty mql means an explanation-only envelope, or one holding
// the prompt's placeholder, never shadows a real answer.
func lastEnvelopeWithMQL(candidates []string) (GenResult, bool) {
	var res GenResult
	found := false
	for _, c := range candidates {
		env, ok := decodeEnvelope(c)
		if !ok {
			continue
		}
		mql := strings.TrimSpace(env.MQL)
		if mql == "" || !plausibleMQL(mql) {
			continue
		}
		res = GenResult{
			// model-authored text reaches a terminal at the review gate
			MQL:         SanitizeModelText(mql),
			Explanation: SanitizeModelText(strings.TrimSpace(env.Explanation)),
		}
		found = true
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

// fencedBlock is one ```<tag> ... ``` block, keeping the tag so the caller can
// tell an untagged block from a ```bash one.
type fencedBlock struct {
	tag  string
	body string
}

// fencedBlocks returns the contents of all ```<lang> ... ``` blocks. An empty
// lang matches a bare ``` fence (any or no language tag).
func fencedBlocks(s, lang string) []string {
	blocks := fencedBlocksMatching(s, lang)
	out := make([]string, 0, len(blocks))
	for _, b := range blocks {
		out = append(out, b.body)
	}
	return out
}

func fencedBlocksMatching(s, lang string) []fencedBlock {
	var out []fencedBlock
	lines := strings.Split(s, "\n")
	inBlock := false
	openTag := ""
	var buf []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !inBlock {
			if strings.HasPrefix(trimmed, "```") {
				tag := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "```")))
				if lang == "" || tag == lang {
					inBlock = true
					openTag = tag
					buf = nil
				}
			}
			continue
		}
		if strings.HasPrefix(trimmed, "```") {
			out = append(out, fencedBlock{tag: openTag, body: strings.Join(buf, "\n")})
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
