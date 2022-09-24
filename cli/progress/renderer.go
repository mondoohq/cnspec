package progress

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/muesli/ansi"
	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/cli/theme/colors"
)

var color func(string) termenv.Color = colors.Profile.Color

// renderer stores values we'll use when rendering the progress bar.
type renderer struct {

	// Total width of the progress bar, including percentage, if set.
	Width int

	// "Filled" sections of the progress bar
	Full      rune
	FullColor string

	// "Empty" sections of progress bar
	Empty      rune
	EmptyColor string

	// Settings for rendering the numeric percentage
	ShowPercentage  bool
	PercentFormat   string // a fmt string for a float
	PercentageStyle *termenv.Style

	useRamp    bool
	rampColorA colorful.Color
	rampColorB colorful.Color

	// When true, we scale the gradient to fit the width of the filled section
	// of the progress bar. When false, the width of the gradient will be set
	// to the full width of the progress bar.
	scaleRamp bool
}

// newRenderer returns a model with default values.
func newRenderer() (*renderer, error) {
	m := &renderer{
		Width:          40,
		Full:           '█',
		FullColor:      "#7571F9",
		Empty:          '░',
		EmptyColor:     "#606060",
		ShowPercentage: true,
		PercentFormat:  " %3.0f%%",
	}

	if err := m.setRamp("#5A56E0", "#EE6FF8", true); err != nil {
		return nil, errors.New("default color setup failed, please report this issue!")
	}

	return m, nil
}

// View renders the progress bar as a given percentage.
func (m *renderer) View(percent float32) string {
	b := strings.Builder{}
	if m.ShowPercentage {
		percentage := fmt.Sprintf(m.PercentFormat, percent*100)
		if m.PercentageStyle != nil {
			percentage = m.PercentageStyle.Styled(percentage)
		}
		m.bar(&b, percent, ansi.PrintableRuneWidth(percentage))
		b.WriteString(percentage)
	} else {
		m.bar(&b, percent, 0)
	}
	return b.String()
}

func (m *renderer) bar(b *strings.Builder, percent float32, textWidth int) {
	var (
		tw = m.Width - textWidth        // total width
		fw = int(float32(tw) * percent) // filled width
		p  float64
	)

	if m.useRamp {
		// Gradient fill
		for i := 0; i < fw; i++ {
			if m.scaleRamp {
				p = float64(i) / float64(fw)
			} else {
				p = float64(i) / float64(tw)
			}
			c := m.rampColorA.BlendLuv(m.rampColorB, p).Hex()
			b.WriteString(termenv.
				String(string(m.Full)).
				Foreground(color(c)).
				String(),
			)
		}
	} else {
		// Solid fill
		s := termenv.String(string(m.Full)).Foreground(color(m.FullColor)).String()
		b.WriteString(strings.Repeat(s, fw))
	}

	// Empty fill
	e := termenv.String(string(m.Empty)).Foreground(color(m.EmptyColor)).String()
	b.WriteString(strings.Repeat(e, tw-fw))
}

func (m *renderer) setRamp(colorA, colorB string, scaled bool) error {
	a, err := colorful.Hex(colorA)
	if err != nil {
		return err
	}

	b, err := colorful.Hex(colorB)
	if err != nil {
		return err
	}

	m.useRamp = true
	m.scaleRamp = scaled
	m.rampColorA = a
	m.rampColorB = b
	return nil
}
