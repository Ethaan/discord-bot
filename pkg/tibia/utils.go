package tibia

import "strings"

func VocationEmoji(vocation string) string {
	switch strings.ToLower(vocation) {
	case "master sorcerer", "sorcerer":
		return "🧙‍♂️"
	case "elder druid", "druid":
		return "🌿"
	case "royal paladin", "paladin":
		return "🏹"
	case "elite knight", "knight":
		return "⚔️"
	default:
		return "❓"
	}
}
