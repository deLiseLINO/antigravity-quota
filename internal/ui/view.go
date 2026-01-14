package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"

	"github.com/deLiseLINO/antigravity-quota/internal/api"
)

func (m Model) View() string {
	var s strings.Builder

	s.WriteString(TitleStyle.Render("🚀 Antigravity API Quota Monitor"))
	s.WriteString("\n")

	if len(m.Tokens) > 0 {
		var tabs []string
		for i, token := range m.Tokens {
			email := token.Email
			if email == "" {
				email = "Unknown"
			}
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			if i == m.ActiveTokenIdx {
				style = style.Foreground(lipgloss.Color("255")).Bold(true).Underline(true)
			}
			tabs = append(tabs, style.Render(email))
		}
		s.WriteString(strings.Join(tabs, " • "))
		s.WriteString("\n\n")
	}

	if m.DeleteConfirm {
		s.WriteString(ErrorStyle.Render("Are you sure you want to delete this account? [x] Confirm [esc] Cancel"))
		s.WriteString("\n\n")
	}

	if m.Loading {
		s.WriteString("Loading...")
		s.WriteString("\n")
	} else if m.LoggingIn {
		s.WriteString("Waiting for authentication in browser...")
		s.WriteString("\n")
	} else if m.Err != nil {
		s.WriteString(ErrorStyle.Render(fmt.Sprintf("Error: %v", m.Err)))
		s.WriteString("\n")
	} else {
		switch m.Mode {
		case ModeGroups:
			s.WriteString(m.renderGroupsView())
		case ModeAll:
			s.WriteString(m.renderAllView())
		}
	}

	s.WriteString(HelpStyle.Render("\n[r] refresh • [tab/m] mode • [n] add account • [x] delete • [←/→] switch • [q] quit"))

	content := s.String()

	contentWidth := lipgloss.Width(content)
	contentHeight := lipgloss.Height(content)

	containerStyle := lipgloss.NewStyle().Padding(1, 2)

	if m.Width > contentWidth+4 && m.Height > contentHeight+2 {
		containerStyle = containerStyle.
			Width(m.Width).
			Height(m.Height).
			Align(lipgloss.Center, lipgloss.Center)
	}

	return containerStyle.Render(content)
}

func (m Model) renderGroupsView() string {
	var s strings.Builder

	s.WriteString(GroupHeaderStyle.Render("Claude"))
	s.WriteString("\n")
	s.WriteString(m.renderModelRow(m.Data.Claude, m.claudeProgress))
	s.WriteString("\n")

	s.WriteString(GroupHeaderStyle.Render("Gemini"))
	s.WriteString("\n")
	s.WriteString(m.renderModelRow(m.Data.Gemini, m.geminiProgress))
	s.WriteString("\n")

	return s.String()
}

func (m Model) renderAllView() string {
	var s strings.Builder

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
			s.WriteString(m.renderModelRow(mq, m.claudeProgress))
		}
	}

	if len(geminiModels) > 0 {
		s.WriteString("\n")
		s.WriteString(GeminiSubheaderStyle.Render("  ▸ Gemini"))
		s.WriteString("\n")
		for _, mq := range geminiModels {
			s.WriteString(m.renderModelRow(mq, m.geminiProgress))
		}
	}

	if len(otherModels) > 0 {
		s.WriteString("\n")
		s.WriteString(OtherSubheaderStyle.Render("  ▸ Other"))
		s.WriteString("\n")
		for _, mq := range otherModels {
			s.WriteString(m.renderModelRow(mq, m.geminiProgress))
		}
	}

	return s.String()
}

func (m Model) renderModelRow(mq api.ModelQuota, prog progress.Model) string {
	var s strings.Builder

	percent := mq.Quota * 100
	name := mq.Name
	if len(name) > 33 {
		name = name[:30] + "..."
	}

	alignedName := fmt.Sprintf("%-35s", name)

	s.WriteString("    ")
	s.WriteString(LabelStyle.Render(alignedName))
	s.WriteString(" ")
	s.WriteString(prog.ViewAs(mq.Quota))
	s.WriteString(" ")
	s.WriteString(PercentStyle.Render(fmt.Sprintf("%.1f%%", percent)))

	if !mq.ResetTime.IsZero() {
		remaining := time.Until(mq.ResetTime)
		if remaining > 0 {
			resetStr := ""
			if remaining.Hours() >= 1 {
				resetStr = fmt.Sprintf("reset in %.1fh", remaining.Hours())
			} else {
				resetStr = fmt.Sprintf("reset in %.0fm", remaining.Minutes())
			}
			s.WriteString(ResetTimeStyle.Render(resetStr))
		}
	} else {
		s.WriteString(ResetTimeStyle.Render("reset in unknown"))
	}

	s.WriteString("\n")

	return s.String()
}
