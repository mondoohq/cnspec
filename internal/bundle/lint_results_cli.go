package bundle

import (
	"bytes"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/olekukonko/tablewriter"
)

type SortResults []Entry

func (s SortResults) Len() int {
	return len(s)
}

func (s SortResults) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s SortResults) Less(i, j int) bool {
	if s[i].Level != s[j].Level {
		return s[i].Level < s[j].Level
	}
	return s[i].RuleID < s[j].RuleID
}

func (r *Results) ToCli() []byte {
	sort.Sort(SortResults(r.Entries))

	// render platform advisories
	var buf bytes.Buffer
	table := tablewriter.NewWriter(&buf)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetBorder(false)
	table.SetHeaderLine(false)
	table.SetRowLine(false)
	table.SetColumnSeparator("")
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	header := []string{"Rule ID", "Level", "File", "Line", "Message"}
	table.SetHeader(header)

	for i := range r.Entries {
		entry := r.Entries[i]
		table.Append([]string{
			entry.RuleID,
			entry.Level,
			filepath.Base(entry.Location[0].File),
			strconv.Itoa(entry.Location[0].Line),
			entry.Message,
		})
	}
	table.Render()
	return buf.Bytes()
}
