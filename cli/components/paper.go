// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package components

import (
	"strings"

	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/v11/cli/theme/colors"
)

type PaperCharsTheme struct {
	Background  rune
	Horizontal  rune
	Vertical    rune
	TopLeft     rune
	TopRight    rune
	BottomLeft  rune
	BottomRight rune
}

var (
	AsciiPaperCars = PaperCharsTheme{
		Background:  ' ',
		Horizontal:  '-',
		Vertical:    '|',
		TopLeft:     '+',
		TopRight:    '+',
		BottomLeft:  '+',
		BottomRight: '+',
	}

	AnsiPaperCars = PaperCharsTheme{
		Background:  ' ',
		Horizontal:  '─',
		Vertical:    '│',
		TopLeft:     '┌',
		TopRight:    '┐',
		BottomLeft:  '└',
		BottomRight: '┘',
	}
)

type PaperOption func(chart *Paper)

func WithPaperCharacterTheme(theme PaperCharsTheme) PaperOption {
	return func(p *Paper) {
		p.theme = theme
	}
}

func NewPaper(opts ...PaperOption) *Paper {
	theme := AnsiPaperCars
	if colors.Profile == termenv.Ascii {
		theme = AsciiPaperCars
	}

	p := &Paper{
		theme: theme,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

type Paper struct {
	theme PaperCharsTheme
}

func (p *Paper) Render(width int, height int) string {
	b := strings.Builder{}

	for i := 0; i < height; i++ {
		firstChar := p.theme.Vertical
		if i == 0 {
			firstChar = p.theme.TopLeft
		} else if i == height-1 {
			firstChar = p.theme.BottomLeft
		}
		b.WriteRune(firstChar)

		horizontal := string(p.theme.Background)
		// first and last line use horizontal char
		if i == 0 || i == height-1 {
			horizontal = string(p.theme.Horizontal)
		}
		b.WriteString(strings.Repeat(horizontal, width-2))

		lastChar := p.theme.Vertical
		if i == 0 {
			lastChar = p.theme.TopRight
		} else if i == height-1 {
			lastChar = p.theme.BottomRight
		}
		b.WriteRune(lastChar)
		b.WriteByte('\n')
	}

	return b.String()
}
