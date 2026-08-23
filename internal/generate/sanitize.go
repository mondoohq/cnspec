// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package generate

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// SanitizeModelText makes model-authored text safe to print at the human review
// gate. Everything the agent returns — the MQL, the explanation, its stdout and
// stderr — is attacker-reachable text (a check's title and description become
// the prompt), and the accept/edit/regenerate loop is the control this design
// leans on. A terminal renders control characters instead of showing them, so
// raw output lets the reviewed party choose what the reviewer sees: a query
// carrying ESC[2K ESC[1G displays as one harmless-looking line while the string
// actually stored in the bundle contains something else entirely.
//
// It escapes rather than deletes, so nothing is silently dropped: the reviewer
// sees `\x1b[2K` and knows the model sent an escape sequence. Newline and tab
// survive as themselves because MQL uses them and they cannot repaint a line.
//
// The characters it neutralizes:
//   - C0 controls (except \n and \t), DEL and the C1 range — cursor movement,
//     line erase, carriage return, and terminal-mode switches.
//   - Unicode bidi overrides and invisible marks (the "trojan source" class),
//     which reorder a line's rendering without changing its bytes.
//   - Invalid UTF-8 bytes, which some terminals interpret as their own commands.
func SanitizeModelText(s string) string {
	if !needsSanitizing(s) {
		return s // the overwhelmingly common case: return the input unchanged
	}

	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			// not valid UTF-8; show the raw byte rather than pass it through
			b.WriteString(`\x` + hexByte(s[i]))
			i++
			continue
		}
		switch {
		case r == '\n' || r == '\t':
			b.WriteRune(r)
		case r < 0x20 || r == 0x7f:
			b.WriteString(`\x` + hexByte(byte(r)))
		case r >= 0x80 && r <= 0x9f:
			b.WriteString(`\u` + pad4(strconv.FormatInt(int64(r), 16)))
		case isInvisibleFormatting(r):
			b.WriteString(`\u` + pad4(strconv.FormatInt(int64(r), 16)))
		default:
			b.WriteRune(r)
		}
		i += size
	}
	return b.String()
}

// needsSanitizing reports whether s contains anything SanitizeModelText would
// rewrite. Scanning first keeps the common path allocation-free.
func needsSanitizing(s string) bool {
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			return true
		}
		if r != '\n' && r != '\t' && (r < 0x20 || r == 0x7f || (r >= 0x80 && r <= 0x9f) || isInvisibleFormatting(r)) {
			return true
		}
		i += size
	}
	return false
}

// isInvisibleFormatting covers the bidi overrides and invisible marks that
// change how a line renders without changing what it says — the trojan-source
// trick. They have no place in MQL or in a one-line explanation.
func isInvisibleFormatting(r rune) bool {
	switch {
	case r == 0x200b, r == 0x200c, r == 0x200d: // zero width space/non-joiner/joiner
		return true
	case r == 0x200e, r == 0x200f: // LTR/RTL marks
		return true
	case r >= 0x202a && r <= 0x202e: // bidi embedding/override
		return true
	case r >= 0x2066 && r <= 0x2069: // bidi isolates
		return true
	case r == 0xfeff: // zero width no-break space / BOM
		return true
	default:
		return false
	}
}

func hexByte(b byte) string {
	const hex = "0123456789abcdef"
	return string([]byte{hex[b>>4], hex[b&0x0f]})
}

func pad4(s string) string {
	for len(s) < 4 {
		s = "0" + s
	}
	return s
}
