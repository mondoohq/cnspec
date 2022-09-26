package components

import (
	"github.com/muesli/termenv"
	"go.mondoo.com/cnquery/cli/theme/colors"
	cvss_proto "go.mondoo.com/cnquery/resources/packs/core/vadvisor/cvss"
)

func NewCvssIndicator() CvssIndicator {
	theme := colors.DefaultColorTheme

	cvssRatingColorMapping := map[cvss_proto.Severity]termenv.Color{
		cvss_proto.None:     theme.Good,
		cvss_proto.Low:      theme.Low,
		cvss_proto.Medium:   theme.Medium,
		cvss_proto.High:     theme.High,
		cvss_proto.Critical: theme.Critical,
		cvss_proto.Unknown:  theme.Unknown,
	}

	return CvssIndicator{
		indicatorChar:          '■',
		cvssRatingColorMapping: cvssRatingColorMapping,
	}
}

type CvssIndicator struct {
	indicatorChar rune

	// colors for cvss ratings
	cvssRatingColorMapping map[cvss_proto.Severity]termenv.Color
}

func (ci CvssIndicator) Render(severity cvss_proto.Severity) string {
	return termenv.String(string(ci.indicatorChar)).Foreground(ci.rating(severity)).String()
}

func (ci CvssIndicator) rating(r cvss_proto.Severity) termenv.Color {
	c, ok := ci.cvssRatingColorMapping[r]
	if ok {
		return c
	}
	return ci.cvssRatingColorMapping[cvss_proto.Unknown]
}
