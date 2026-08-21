package ui

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/deLiseLINO/antigravity-quota/internal/api"
)

const windowRowIndent = "    "

var partialBarBlocks = [...]string{"", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}

const (
	defaultBarGradientStart = "#6C63FF"
	defaultBarGradientEnd   = "#D46DFF"
	shortBarGradientStart   = "#4285F4"
	shortBarGradientEnd     = "#34A853"
)

func formatResetText(resetAt time.Time) string {
	if resetAt.IsZero() {
		return "Resets unknown"
	}

	remaining := time.Until(resetAt)
	if remaining <= 0 {
		return "Resets now"
	}

	localReset := resetAt.Local()
	now := time.Now().Local()
	absolute := ""

	if sameDay(localReset, now) {
		absolute = localReset.Format("15:04")
	} else if remaining <= 7*24*time.Hour {
		absolute = localReset.Format("Mon 15:04")
	} else {
		absolute = localReset.Format("01-02 15:04")
	}

	return fmt.Sprintf("Resets %s (%s)", absolute, formatRemainingShort(remaining))
}

func formatRemainingShort(remaining time.Duration) string {
	if remaining <= 0 {
		return "now"
	}

	if remaining < time.Minute {
		return "<1m"
	}

	totalMinutes := int(remaining.Minutes())
	if totalMinutes < 60 {
		return fmt.Sprintf("%dm", totalMinutes)
	}

	totalHours := int(remaining.Hours())
	if totalHours < 24 {
		mins := totalMinutes % 60
		if mins == 0 {
			return fmt.Sprintf("%dh", totalHours)
		}
		return fmt.Sprintf("%dh %dm", totalHours, mins)
	}

	days := totalHours / 24
	hours := totalHours % 24
	if hours == 0 {
		return fmt.Sprintf("%dd", days)
	}
	return fmt.Sprintf("%dd %dh", days, hours)
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func clampRatio(ratio float64) float64 {
	if ratio < 0 {
		return 0
	}
	if ratio > 1 {
		return 1
	}
	return ratio
}

func renderSmoothBar(width int, ratio float64, startHex string, endHex string) string {
	if width <= 0 {
		width = 40
	}

	ratio = clampRatio(ratio)
	total := ratio * float64(width)
	fullCells := int(math.Floor(total))
	if fullCells < 0 {
		fullCells = 0
	}
	if fullCells > width {
		fullCells = width
	}

	fractional := total - float64(fullCells)
	partialIndex := int(math.Round(fractional * 8))
	if partialIndex >= len(partialBarBlocks) {
		partialIndex = 0
		fullCells++
	}
	if fullCells > width {
		fullCells = width
		partialIndex = 0
	}
	if fullCells == width {
		partialIndex = 0
	}

	var b strings.Builder
	if fullCells > 0 {
		for i := range fullCells {
			cellStyle := gradientCellStyle(startHex, endHex, i, width)
			b.WriteString(cellStyle.Render("█"))
		}
	}

	usedCells := fullCells
	if partialIndex > 0 && usedCells < width {
		cellStyle := gradientCellStyle(startHex, endHex, usedCells, width)
		b.WriteString(cellStyle.Render(partialBarBlocks[partialIndex]))
		usedCells++
	}

	if usedCells < width {
		b.WriteString(BarEmptyStyle.Render(strings.Repeat("·", width-usedCells)))
	}

	return b.String()
}

func barGradientForBucket(b api.QuotaBucket) (string, string) {
	if strings.HasPrefix(strings.ToLower(b.BucketID), "gemini") {
		return shortBarGradientStart, shortBarGradientEnd
	}
	return defaultBarGradientStart, defaultBarGradientEnd
}

func barGradientForModel(name string) (string, string) {
	if api.IsGeminiModel(name) {
		return shortBarGradientStart, shortBarGradientEnd
	}
	return defaultBarGradientStart, defaultBarGradientEnd
}

func gradientCellStyle(startHex string, endHex string, pos int, width int) lipgloss.Style {
	if width <= 1 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(startHex))
	}
	t := float64(pos) / float64(width-1)
	return lipgloss.NewStyle().Foreground(lipgloss.Color(interpolateHexColor(startHex, endHex, t)))
}

func interpolateHexColor(startHex string, endHex string, t float64) string {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	sr, sg, sb := parseHexColor(startHex)
	er, eg, eb := parseHexColor(endHex)
	r := int(math.Round(float64(sr) + (float64(er)-float64(sr))*t))
	g := int(math.Round(float64(sg) + (float64(eg)-float64(sg))*t))
	b := int(math.Round(float64(sb) + (float64(eb)-float64(sb))*t))
	return fmt.Sprintf("#%02X%02X%02X", clampColor(r), clampColor(g), clampColor(b))
}

func parseHexColor(value string) (int, int, int) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(trimmed) != 6 {
		return 255, 255, 255
	}
	parsed, err := strconv.ParseUint(trimmed, 16, 32)
	if err != nil {
		return 255, 255, 255
	}
	r := int((parsed >> 16) & 0xFF)
	g := int((parsed >> 8) & 0xFF)
	b := int(parsed & 0xFF)
	return r, g, b
}

func clampColor(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}
