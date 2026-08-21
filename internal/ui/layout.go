package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func renderMessageModal(title, message string, titleStyle lipgloss.Style, viewportWidth int) string {
	message = strings.TrimSpace(message)
	width := messageModalWidth(title, message, viewportWidth)
	bodyWidth := width - 2
	if bodyWidth < 1 {
		bodyWidth = 1
	}
	wrappedMessage := lipgloss.NewStyle().Width(bodyWidth).Render(message)
	content := strings.Join([]string{
		titleStyle.Render(title),
		InfoValueStyle.Render(wrappedMessage),
	}, "\n\n")
	return InfoBoxStyle.Copy().Width(width).Render(content)
}

func messageModalWidth(title, message string, viewportWidth int) int {
	target := messageModalMinWidth
	for _, line := range strings.Split(strings.TrimSpace(message), "\n") {
		width := ansi.StringWidth(line) + 2
		if width > target {
			target = width
		}
	}
	if titleWidth := ansi.StringWidth(title) + 2; titleWidth > target {
		target = titleWidth
	}
	if target > messageModalMaxWidth {
		target = messageModalMaxWidth
	}
	if viewportWidth <= 0 {
		return target
	}

	maxAllowed := viewportWidth - messageModalInset - 2
	if maxAllowed <= 0 {
		return messageModalMinWidth
	}
	if maxAllowed < messageModalMinWidth {
		return maxAllowed
	}
	if target > maxAllowed {
		return maxAllowed
	}
	return target
}

func overlayCenter(base, modal string, width, height int) string {
	canvasWidth := width
	if canvasWidth < lipgloss.Width(base) {
		canvasWidth = lipgloss.Width(base)
	}
	if canvasWidth < lipgloss.Width(modal)+2 {
		canvasWidth = lipgloss.Width(modal) + 2
	}

	canvasHeight := height
	if canvasHeight < lipgloss.Height(base) {
		canvasHeight = lipgloss.Height(base)
	}
	if canvasHeight < lipgloss.Height(modal)+2 {
		canvasHeight = lipgloss.Height(modal) + 2
	}

	baseCanvas := lipgloss.Place(canvasWidth, canvasHeight, lipgloss.Left, lipgloss.Top, base)
	baseLines := strings.Split(baseCanvas, "\n")
	modalLines := strings.Split(modal, "\n")

	modalWidth := lipgloss.Width(modal)
	modalHeight := len(modalLines)
	startX := (canvasWidth - modalWidth) / 2
	if startX < 0 {
		startX = 0
	}
	startY := (canvasHeight - modalHeight) / 2
	if startY < 0 {
		startY = 0
	}

	for i, modalLine := range modalLines {
		y := startY + i
		if y < 0 || y >= len(baseLines) {
			continue
		}

		line := padANSI(baseLines[y], canvasWidth)
		modalLine = padANSI(modalLine, modalWidth)

		left := ansi.Cut(line, 0, startX)
		right := ansi.Cut(line, startX+modalWidth, canvasWidth)
		baseLines[y] = left + modalLine + right
	}

	return strings.Join(baseLines, "\n")
}

func padANSI(line string, targetWidth int) string {
	currentWidth := ansi.StringWidth(line)
	if currentWidth >= targetWidth {
		return line
	}
	return line + strings.Repeat(" ", targetWidth-currentWidth)
}

func splitFooterArea(view string, footerHeight int) (string, string) {
	if footerHeight <= 0 {
		return view, ""
	}
	lines := strings.Split(view, "\n")
	if footerHeight >= len(lines) {
		return view, ""
	}
	body := strings.Join(lines[:len(lines)-footerHeight], "\n")
	footer := strings.Join(lines[len(lines)-footerHeight:], "\n")
	return body, footer
}

func joinFooterArea(body, footer string) string {
	if strings.TrimSpace(footer) == "" {
		return body
	}
	if body == "" {
		return footer
	}
	return body + "\n" + footer
}
