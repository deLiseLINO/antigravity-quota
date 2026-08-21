package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	messageModalMinWidth = 64
	messageModalMaxWidth = 104
	messageModalInset    = 6
)

func (m Model) currentOverlayModal() string {
	if m.HelpVisible {
		return m.renderHelpModal()
	}

	if m.DeleteConfirm {
		return m.renderDeleteConfirmModal()
	}

	if m.LoggingIn {
		return renderMessageModal(
			"Waiting for authentication",
			"Complete authorization in your browser.",
			InfoTitleStyle,
			m.Width,
		)
	}

	if m.Err != nil {
		return m.renderErrorModal()
	}

	if m.Notice != "" {
		return renderMessageModal("Notice", m.Notice, NoticeStyle, m.Width)
	}

	if len(m.Tokens) == 0 {
		return renderMessageModal("No accounts", "No accounts loaded.\nPress n to add account.", WarningStyle, m.Width)
	}

	return ""
}

func (m Model) renderErrorModal() string {
	message := strings.TrimSpace(m.Err.Error())
	if message == "" {
		message = "Unknown error"
	}
	hint := "[enter/esc] Close"
	width := messageModalWidth("Error", message+"\n"+hint, m.Width)
	bodyWidth := width - 2
	if bodyWidth < 1 {
		bodyWidth = 1
	}
	wrappedMessage := lipgloss.NewStyle().Width(bodyWidth).Render(message)
	content := strings.Join([]string{
		ErrorStyle.Render("Error"),
		InfoValueStyle.Render(wrappedMessage),
		InfoValueStyle.Render(hint),
	}, "\n\n")
	return InfoBoxStyle.Copy().Width(width).Render(content)
}

func (m Model) renderDeleteConfirmModal() string {
	lines := []string{
		WarningStyle.Render("Delete account"),
		InfoValueStyle.Render("Are you sure you want to delete this account?"),
		InfoValueStyle.Render("[x] Confirm   [esc] Cancel"),
	}
	content := strings.Join(lines, "\n")
	return InfoBoxStyle.Copy().Width(68).Render(content)
}

func (m Model) renderHelpModal() string {
	lines := []string{
		InfoTitleStyle.Render("Keyboard help"),
		"",
		HelpSectionStyle.Render("Account"),
		renderHelpLine("r", "Refresh"),
		renderHelpLine("n", "Add account"),
		renderHelpLine("x", "Delete account"),
		renderHelpLine("←/→ / h/l", "Switch account"),
		"",
		HelpSectionStyle.Render("Other"),
		renderHelpLine("tab / m", "Toggle Groups / All"),
		renderHelpLine("?", "Open or close this help"),
		renderHelpLine("q", "Quit"),
		"",
	}
	return InfoBoxStyle.Copy().Width(56).Render(strings.Join(lines, "\n"))
}

func renderHelpLine(key, description string) string {
	return fmt.Sprintf("%s %s", HelpKeyStyle.Render(fmt.Sprintf("%-10s", key)), InfoValueStyle.Render(description))
}
