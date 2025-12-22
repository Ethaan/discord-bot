package ascii

import (
	"bytes"
	"fmt"

	"github.com/ethaan/discord-api/pkg/repositories"
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

	table.Header("Lvl", "Name", "EXP+")

	totalExp := 0
	for _, pg := range powergamers {
		totalExp = totalExp + pg.Today
		table.Append([]string{
			fmt.Sprintf("%d", pg.Level),
			fmt.Sprintf("%s %s", tibia.VocationEmoji(pg.Vocation), pg.Name),
			tibia.FormatTibiaNumber(pg.Today),
		})
	}

	table.Footer([]string{
		"",
		"Total EXP",
		tibia.FormatTibiaNumber(totalExp),
	})

	table.Render()
	return buf.String()
}

func BuildScanResultsTable(results []repositories.ScanResult, veryHighThreshold, highThreshold, mediumThreshold int) string {
	buf := new(bytes.Buffer)

	table := tablewriter.NewWriter(buf)
	table.Options(
		tablewriter.WithRowAutoWrap(0),
		tablewriter.WithRowAlignment(tw.AlignLeft),
	)

	table.Header("Character Name", "Transitions", "Confidence")

	for _, r := range results {
		var confidence string
		var emoji string

		if r.AdjacentCount >= veryHighThreshold {
			emoji = "🔴"
			confidence = "Very High"
		} else if r.AdjacentCount >= highThreshold {
			emoji = "🟠"
			confidence = "High"
		} else if r.AdjacentCount >= mediumThreshold {
			emoji = "🟡"
			confidence = "Medium"
		} else {
			emoji = "⚪"
			confidence = "Low"
		}

		table.Append([]string{
			r.CharacterName,
			fmt.Sprintf("%d", r.AdjacentCount),
			fmt.Sprintf("%s %s", emoji, confidence),
		})
	}

	table.Render()
	return buf.String()
}
