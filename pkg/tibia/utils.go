package tibia

import (
	"fmt"
	"strings"
)

func VocationEmoji(vocation string) string {
	switch strings.ToLower(vocation) {
	case "master sorcerer", "sorcerer":
		return "🧙‍♂️ "
	case "elder druid", "druid":
		return "🌿 "
	case "royal paladin", "paladin":
		return "🏹 "
	case "elite knight", "knight":
		return "⚔️ "
	default:
		return "❓ "
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
