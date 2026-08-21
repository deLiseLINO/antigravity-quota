package ui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/deLiseLINO/antigravity-quota/internal/api"
)

func (m Model) View() string {
	var s strings.Builder
	modal := m.currentOverlayModal()

	s.WriteString(TitleStyle.Render("🚀 Antigravity Quota"))
	s.WriteString("\n")

	if len(m.Tokens) > 0 {
		s.WriteString(m.renderAccountTabs())
		s.WriteString("\n\n")
	}

	if m.Loading && len(m.Tokens) > 0 && !m.hasCachedDisplayData() {
		s.WriteString(m.renderGroupsLoadingSkeleton())
	} else {
		switch m.Mode {
		case ModeGroups:
			s.WriteString(m.renderGroupsView())
		case ModeAll:
			s.WriteString(m.renderAllView())
		}
	}

	footer := HelpStyle.Render("\n" + m.renderFooter())
	s.WriteString(footer)

	content := s.String()
	containerStyle := lipgloss.NewStyle().Padding(1, 2)
	hAlign := lipgloss.Left
	vAlign := lipgloss.Top
	if m.Width > 0 {
		containerStyle = containerStyle.Width(m.Width)
		hAlign = lipgloss.Center
	}
	if m.Height > 0 {
		containerStyle = containerStyle.Height(m.Height)
		vAlign = lipgloss.Center
	}
	containerStyle = containerStyle.Align(hAlign, vAlign)

	baseView := containerStyle.Render(content)

	if modal != "" {
		body, footerArea := splitFooterArea(baseView, lipgloss.Height(footer))
		return joinFooterArea(overlayCenter(body, modal, m.Width, m.Height-lipgloss.Height(footer)), footerArea)
	}

	return baseView
}

func (m Model) preferredContentWidth() int {
	if m.Width <= 0 {
		return 0
	}
	if m.Width <= 12 {
		return m.Width
	}
	usable := m.Width - 4
	const maxContentWidth = 220
	if usable > maxContentWidth {
		return maxContentWidth
	}
	return usable
}

func (m Model) renderFooter() string {
	return "←→ Switch • r Refresh • tab Mode • n Add • x Delete • ? Help • q Quit"
}

func (m Model) activeTokenKey() string {
	if m.ActiveTokenIdx < 0 || m.ActiveTokenIdx >= len(m.Tokens) {
		return ""
	}
	return tokenKey(m.Tokens[m.ActiveTokenIdx])
}

func (m Model) hasCachedDisplayData() bool {
	if m.ActiveTokenIdx < 0 || m.ActiveTokenIdx >= len(m.Tokens) {
		return false
	}
	key := tokenKey(m.Tokens[m.ActiveTokenIdx])
	if key == "" {
		return false
	}
	if m.Mode == ModeGroups {
		_, ok := m.SummaryCache[key]
		return ok
	}
	_, ok := m.DataCache[key]
	return ok
}

func (m Model) renderGroupsView() string {
	var s strings.Builder
	accountKey := m.activeTokenKey()
	for _, group := range m.Summary.Groups {
		s.WriteString(GroupHeaderStyle.Render(group.DisplayName))
		s.WriteString("\n")
		buckets := make([]api.QuotaBucket, 0, len(group.Buckets))
		for _, bucket := range group.Buckets {
			if bucket.Disabled {
				continue
			}
			buckets = append(buckets, bucket)
		}
		sort.SliceStable(buckets, func(i, j int) bool {
			return bucketOrder(buckets[i]) < bucketOrder(buckets[j])
		})
		for _, bucket := range buckets {
			s.WriteString(m.renderBucketRow(bucket, accountKey))
			s.WriteString("\n")
		}
	}
	return s.String()
}

func (m Model) renderGroupsLoadingSkeleton() string {
	var s strings.Builder
	s.WriteString(GroupHeaderStyle.Render("Loading..."))
	s.WriteString("\n")
	for range 3 {
		s.WriteString(m.renderSkeletonRow())
		s.WriteString("\n")
	}
	return s.String()
}

func bucketLabel(b api.QuotaBucket) string {
	if isFiveHourBucket(b) {
		return "5 hour"
	}
	if isWeeklyBucket(b) {
		return "Weekly"
	}
	name := b.DisplayName
	if name == "" {
		name = b.Window
	}
	if name == "" {
		name = b.BucketID
	}
	return name
}

func isFiveHourBucket(b api.QuotaBucket) bool {
	haystack := strings.ToLower(b.BucketID + " " + b.Window + " " + b.DisplayName)
	return strings.Contains(haystack, "5h") ||
		strings.Contains(haystack, "5 h") ||
		strings.Contains(haystack, "five hour")
}

func isWeeklyBucket(b api.QuotaBucket) bool {
	haystack := strings.ToLower(b.BucketID + " " + b.Window + " " + b.DisplayName)
	return strings.Contains(haystack, "week")
}

func bucketOrder(b api.QuotaBucket) int {
	switch {
	case isFiveHourBucket(b):
		return 0
	case isWeeklyBucket(b):
		return 1
	default:
		return 2
	}
}

func (m Model) renderBucketRow(b api.QuotaBucket, accountKey string) string {
	var s strings.Builder

	key := bucketAnimationKey(accountKey, b)
	ratio := m.barRatio(key, clampRatio(b.RemainingFraction))

	nameWidth, barWidth, percentWidth, resetWidth := m.rowLayout()
	leadOffset := m.leadOffset()
	name := truncateLabel(bucketLabel(b), nameWidth)
	alignedName := padRight(name, nameWidth)
	percentText := fmt.Sprintf("%.0f%%", b.RemainingFraction*100)
	if ansi.StringWidth(percentText) > percentWidth {
		percentText = truncateLabel(percentText, percentWidth)
	}
	resetText := truncateLabelFromLeft(formatResetText(b.ResetTime), resetWidth)
	gradientStart, gradientEnd := barGradientForBucket(b)

	s.WriteString(strings.Repeat(" ", leadOffset))
	s.WriteString(windowRowIndent)
	s.WriteString(LabelStyle.Render(alignedName))
	s.WriteString(" ")
	s.WriteString(renderSmoothBar(barWidth, ratio, gradientStart, gradientEnd))
	s.WriteString(" ")
	s.WriteString(PercentStyle.Copy().Width(percentWidth).Render(percentText))
	if resetWidth > 0 && strings.TrimSpace(resetText) != "" {
		s.WriteString(ResetTimeStyle.Copy().Width(resetWidth).Render(resetText))
	}

	return s.String()
}

func (m Model) renderSkeletonRow() string {
	var s strings.Builder
	nameWidth, barWidth, percentWidth, resetWidth := m.rowLayout()
	leadOffset := m.leadOffset()
	name := truncateLabel("Loading...", nameWidth)
	alignedName := padRight(name, nameWidth)
	status := truncateLabelStrict("Loading...", resetWidth)

	s.WriteString(strings.Repeat(" ", leadOffset))
	s.WriteString(windowRowIndent)
	s.WriteString(LabelStyle.Render(alignedName))
	s.WriteString(" ")
	s.WriteString(renderSmoothBar(barWidth, 0, defaultBarGradientStart, defaultBarGradientEnd))
	s.WriteString(" ")
	s.WriteString(PercentStyle.Copy().Width(percentWidth).Render("..."))
	if resetWidth > 0 && strings.TrimSpace(status) != "" {
		s.WriteString(ResetTimeStyle.Copy().Width(resetWidth).Render(status))
	}
	return s.String()
}

func (m Model) renderAllView() string {
	var s strings.Builder
	accountKey := m.activeTokenKey()

	s.WriteString(GroupHeaderStyle.Render(fmt.Sprintf("All Models (%d)", len(m.Data.All))))
	s.WriteString("\n")

	var claudeModels, geminiModels, otherModels []api.ModelQuota
	for _, mq := range m.Data.All {
		if api.IsClaudeModel(mq.Name) {
			claudeModels = append(claudeModels, mq)
		} else if api.IsGeminiModel(mq.Name) {
			geminiModels = append(geminiModels, mq)
		} else {
			otherModels = append(otherModels, mq)
		}
	}

	if len(claudeModels) > 0 {
		s.WriteString("\n")
		s.WriteString(ClaudeSubheaderStyle.Render("  ▸ Claude"))
		s.WriteString("\n")
		for _, mq := range claudeModels {
			s.WriteString(m.renderModelRow(mq, accountKey))
		}
	}

	if len(geminiModels) > 0 {
		s.WriteString("\n")
		s.WriteString(GeminiSubheaderStyle.Render("  ▸ Gemini"))
		s.WriteString("\n")
		for _, mq := range geminiModels {
			s.WriteString(m.renderModelRow(mq, accountKey))
		}
	}

	if len(otherModels) > 0 {
		s.WriteString("\n")
		s.WriteString(OtherSubheaderStyle.Render("  ▸ Other"))
		s.WriteString("\n")
		for _, mq := range otherModels {
			s.WriteString(m.renderModelRow(mq, accountKey))
		}
	}

	return s.String()
}

func (m Model) renderModelRow(mq api.ModelQuota, accountKey string) string {
	var s strings.Builder

	key := modelAnimationKey(accountKey, mq)
	ratio := m.barRatio(key, clampRatio(mq.Quota))

	nameWidth, barWidth, percentWidth, resetWidth := m.rowLayout()
	leadOffset := m.leadOffset()
	name := truncateLabel(mq.Name, nameWidth)
	alignedName := padRight(name, nameWidth)
	percentText := fmt.Sprintf("%.0f%%", mq.Quota*100)
	if ansi.StringWidth(percentText) > percentWidth {
		percentText = truncateLabel(percentText, percentWidth)
	}
	resetText := truncateLabelFromLeft(formatResetText(mq.ResetTime), resetWidth)
	gradientStart, gradientEnd := barGradientForModel(mq.Name)

	s.WriteString(strings.Repeat(" ", leadOffset))
	s.WriteString(windowRowIndent)
	s.WriteString(LabelStyle.Render(alignedName))
	s.WriteString(" ")
	s.WriteString(renderSmoothBar(barWidth, ratio, gradientStart, gradientEnd))
	s.WriteString(" ")
	s.WriteString(PercentStyle.Copy().Width(percentWidth).Render(percentText))
	if resetWidth > 0 && strings.TrimSpace(resetText) != "" {
		s.WriteString(ResetTimeStyle.Copy().Width(resetWidth).Render(resetText))
	}

	return s.String()
}

func (m Model) rowLayout() (nameWidth, barWidth, percentWidth, resetWidth int) {
	nameWidth = 22
	barWidth = m.barWidth
	percentWidth = 5
	resetWidth = 26

	if m.Width <= 0 {
		return
	}

	const (
		minNameWidth      = 6
		minNameSoftWidth  = 8
		minBarWidth       = 8
		minBarSoftWidth   = 10
		minPercentWidth   = 4
		minResetWidth     = 0
		minResetSoftWidth = 8
		gapsWidth         = 2
		resetMarginLeft   = 2
	)

	available := m.preferredContentWidth() - ansi.StringWidth(windowRowIndent)
	if available <= 0 {
		return
	}
	switch contentWidth := m.preferredContentWidth(); {
	case contentWidth <= 104 && available > 24:
		available -= 8
	case contentWidth <= 120 && available > 24:
		available -= 4
	}

	used := nameWidth + barWidth + percentWidth + resetWidth + gapsWidth + resetMarginLeft
	shortage := used - available
	if shortage <= 0 {
		return
	}

	reduce := func(current, minimum int) int {
		if shortage <= 0 {
			return current
		}
		canReduce := current - minimum
		if canReduce <= 0 {
			return current
		}
		if canReduce > shortage {
			canReduce = shortage
		}
		shortage -= canReduce
		return current - canReduce
	}

	reduceBalanced := func(left, leftMin, right, rightMin int) (int, int) {
		for shortage > 0 {
			progressed := false
			if left > leftMin {
				left--
				shortage--
				progressed = true
			}
			if shortage > 0 && right > rightMin {
				right--
				shortage--
				progressed = true
			}
			if !progressed {
				break
			}
		}
		return left, right
	}

	nameWidth, resetWidth = reduceBalanced(nameWidth, minNameSoftWidth, resetWidth, minResetSoftWidth)
	barWidth = reduce(barWidth, minBarSoftWidth)
	percentWidth = reduce(percentWidth, minPercentWidth)
	nameWidth, resetWidth = reduceBalanced(nameWidth, minNameWidth, resetWidth, minResetWidth)
	barWidth = reduce(barWidth, minBarWidth)
	return
}

func (m Model) rowDisplayWidth() int {
	nameWidth, barWidth, percentWidth, resetWidth := m.rowLayout()
	const (
		gapsWidth       = 2
		resetMarginLeft = 2
	)
	return ansi.StringWidth(windowRowIndent) + nameWidth + barWidth + percentWidth + resetWidth + gapsWidth + resetMarginLeft
}

func (m Model) leadOffset() int {
	nameWidth, barWidth, _, _ := m.rowLayout()
	rowWidth := m.rowDisplayWidth()
	currentBarCenter := ansi.StringWidth(windowRowIndent) + nameWidth + 1 + (barWidth / 2)

	offset := rowWidth - (2 * currentBarCenter)
	offset += 4
	if offset <= 0 {
		return 0
	}

	maxOffset := m.preferredContentWidth() - rowWidth
	if maxOffset < 0 {
		maxOffset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	return offset
}

func padRight(value string, width int) string {
	if width <= 0 {
		return value
	}
	current := ansi.StringWidth(value)
	if current >= width {
		return value
	}
	return value + strings.Repeat(" ", width-current)
}
