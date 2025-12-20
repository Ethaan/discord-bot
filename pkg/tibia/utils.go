package tibia

import (
	"fmt"
	"strings"
)

func VocationEmoji(vocation string) string {
	switch strings.ToLower(vocation) {
	case "elder druid", "druid":
		return "<:paralyze:1451811045424369715>"
	case "master sorcerer", "sorcerer":
		return "<:sd:1451812855576920186>"
	case "royal paladin", "paladin":
		return "<:crossbow:1451811296667369616>"
	case "elite knight", "knight":
		return "<:magicsword:1451811399545258014>"
	default:
		return ""
	}
}

func FormatTibiaNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	} else if n < 1000000 {
		// Format as k
		if n%1000 == 0 {
			return fmt.Sprintf("%dk", n/1000)
		}
		return fmt.Sprintf("%.1fk", float64(n)/1000.0)
	} else {
		// Format as kk
		if n%1000000 == 0 {
			return fmt.Sprintf("%dkk", n/1000000)
		}
		return fmt.Sprintf("%.1fkk", float64(n)/1000000.0)
	}
}
