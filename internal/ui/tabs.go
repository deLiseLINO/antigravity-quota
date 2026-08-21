package ui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/deLiseLINO/antigravity-quota/internal/config"
)

func (m Model) renderAccountTabs() string {
	tokens := m.Tokens
	activeIndex := m.ActiveTokenIdx
	width := m.Width

	if len(tokens) == 0 {
		return ""
	}

	maxVisible := 3
	switch {
	case width < 60:
		maxVisible = 1
	case width < 78:
		maxVisible = 2
	}

	baseLabelLimit := m.tabLabelLimit()
	for maxVisible >= 1 {
		start, end := tabVisibleRange(len(tokens), activeIndex, maxVisible)
		for labelLimit := baseLabelLimit; labelLimit >= 6; labelLimit-- {
			tabs := m.renderTabsRange(tokens, start, end, activeIndex, labelLimit)
			rendered := strings.Join(tabs, " • ")
			if m.tabsFitsWidth(rendered) || (maxVisible == 1 && labelLimit == 6) {
				return rendered
			}
		}
		maxVisible--
	}

	return ""
}

func tabVisibleRange(total, activeIndex, maxVisible int) (start, end int) {
	start = 0
	end = total
	if total <= maxVisible {
		return start, end
	}

	half := maxVisible / 2
	start = activeIndex - half
	if start < 0 {
		start = 0
	}
	end = start + maxVisible
	if end > total {
		end = total
		start = end - maxVisible
	}
	return start, end
}

func (m Model) renderTabsRange(tokens []*config.TokenFile, start, end, activeIndex, labelLimit int) []string {
	tabs := make([]string, 0, (end-start)+2)
	if start > 0 {
		tabs = append(tabs, TabInactiveStyle.Render("..."))
	}

	for i := start; i < end; i++ {
		token := tokens[i]
		label := token.Email
		if label == "" {
			label = "Unknown"
		}
		label = truncateLabel(label, labelLimit)
		labelText := TabInactiveStyle.Render(label)
		if i == activeIndex {
			labelText = TabActiveStyle.Render(label)
		}
		tabs = append(tabs, labelText)
	}

	if end < len(tokens) {
		tabs = append(tabs, TabInactiveStyle.Render("..."))
	}

	return tabs
}

func (m Model) tabsFitsWidth(rendered string) bool {
	if m.Width <= 0 {
		return true
	}
	available := m.preferredContentWidth() - 2
	if available < 12 {
		available = m.Width
	}
	return ansi.StringWidth(rendered) <= available
}

func (m Model) tabLabelLimit() int {
	width := m.Width
	switch {
	case width >= 180:
		return 28
	case width >= 150:
		return 24
	case width >= 120:
		return 20
	case width >= 100:
		return 16
	case width >= 80:
		return 12
	default:
		return 8
	}
}

func truncateLabel(value string, limit int) string {
	if limit <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	if limit <= 1 {
		return string(runes[:limit])
	}
	return string(runes[:limit-1]) + "…"
}

func truncateLabelFromLeft(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	if limit == 1 {
		return "…"
	}
	return "…" + string(runes[len(runes)-(limit-1):])
}

func truncateLabelStrict(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	return truncateLabel(value, limit)
}
