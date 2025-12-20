package ascii

import (
	"bytes"
	"fmt"

	"github.com/ethaan/discord-api/pkg/tibia"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

func BuildTextTableForPowergamers(powergamers []tibia.Powergamer) string {
	buf := new(bytes.Buffer)

	table := tablewriter.NewWriter(buf)
	table.Options(
		tablewriter.WithRowAutoWrap(0),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)

	table.Header("Lvl", "Name", "EXP+", "Vocation")

	// add rows
	for _, pg := range powergamers {
		table.Append([]string{
			fmt.Sprintf("%d", pg.Level),
			pg.Name,
			tibia.FormatTibiaNumber(pg.Today),
			tibia.VocationEmoji(pg.Vocation),
		})
	}

	table.Render()
	return buf.String()
}
