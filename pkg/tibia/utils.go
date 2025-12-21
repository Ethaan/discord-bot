package tibia

import (
	"fmt"
	"strings"
)

func VocationEmoji(vocation string) string {
	switch strings.ToLower(vocation) {
	case "elite knight", "knight":
		return "[EK]"
	case "elder druid", "druid":
		return "[ED]"
	case "royal paladin", "paladin":
		return "[RP]"
	case "master sorcerer", "sorcerer":
		return "[MS]"
	default:
		return "[NA]"
	}
}

func FormatTibiaNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	} else if n < 1000000 {
		if n%1000 == 0 {
			return fmt.Sprintf("%dk", n/1000)
		}
		return fmt.Sprintf("%.1fk", float64(n)/1000.0)
	} else {
		if n%1000000 == 0 {
			return fmt.Sprintf("%dkk", n/1000000)
		}
		return fmt.Sprintf("%.1fkk", float64(n)/1000000.0)
	}
}
