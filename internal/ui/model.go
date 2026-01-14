package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/deLiseLINO/antigravity-quota/internal/api"
	"github.com/deLiseLINO/antigravity-quota/internal/auth"
	"github.com/deLiseLINO/antigravity-quota/internal/config"
)

type Model struct {
	claudeProgress progress.Model
	geminiProgress progress.Model
	Data           api.AllModelsData
	Loading        bool
	DeleteConfirm  bool
	LoggingIn      bool
	Err            error
	Width          int
	Height         int
	Mode           ViewMode
	Tokens         []*config.TokenFile
	ActiveTokenIdx int
}

func InitialModel(tokens []*config.TokenFile) Model {
	claudeProg := progress.New(progress.WithDefaultGradient(), progress.WithoutPercentage())
	geminiProg := progress.New(progress.WithGradient("#4285F4", "#34A853"), progress.WithoutPercentage())

	return Model{
		claudeProgress: claudeProg,
		geminiProgress: geminiProg,
		Loading:        true,
		Mode:           ModeGroups,
		Tokens:         tokens,
		ActiveTokenIdx: 0,
	}
}

func (m Model) Init() tea.Cmd {
	if len(m.Tokens) > 0 {
		return FetchData(m.Tokens[m.ActiveTokenIdx])
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "x", "delete":
			if len(m.Tokens) > 0 {
				if m.DeleteConfirm {
					token := m.Tokens[m.ActiveTokenIdx]
					if err := os.Remove(token.FilePath); err == nil {
						m.Tokens = append(m.Tokens[:m.ActiveTokenIdx], m.Tokens[m.ActiveTokenIdx+1:]...)
						m.DeleteConfirm = false
						if len(m.Tokens) == 0 {
							m.ActiveTokenIdx = 0
							m.Data = api.AllModelsData{}
						} else {
							if m.ActiveTokenIdx >= len(m.Tokens) {
								m.ActiveTokenIdx = len(m.Tokens) - 1
							}
							return m, FetchData(m.Tokens[m.ActiveTokenIdx])
						}
					}
					m.DeleteConfirm = false
					if len(m.Tokens) > 0 {
						if m.ActiveTokenIdx >= len(m.Tokens) {
							m.ActiveTokenIdx = len(m.Tokens) - 1
						}
						return m, FetchData(m.Tokens[m.ActiveTokenIdx])
					}
					return m, nil
				} else {
					m.DeleteConfirm = true
					return m, nil
				}
			}
			return m, nil
		case "esc":
			if m.DeleteConfirm {
				m.DeleteConfirm = false
				return m, nil
			}
			return m, tea.Quit
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			if len(m.Tokens) == 0 {
				return m, nil
			}
			m.Loading = true
			m.Err = nil
			return m, FetchData(m.Tokens[m.ActiveTokenIdx])
		case "n":
			if !m.LoggingIn {
				m.LoggingIn = true
				return m, LoginCmd
			}
			return m, nil
		case "tab", "m":
			if m.Mode == ModeGroups {
				m.Mode = ModeAll
			} else {
				m.Mode = ModeGroups
			}
			return m, nil
		case "right", "l":
			if len(m.Tokens) > 1 {
				m.ActiveTokenIdx = (m.ActiveTokenIdx + 1) % len(m.Tokens)
				m.Loading = true
				m.Data = api.AllModelsData{}
				return m, FetchData(m.Tokens[m.ActiveTokenIdx])
			}
		case "left", "h":
			if len(m.Tokens) > 1 {
				m.ActiveTokenIdx = (m.ActiveTokenIdx - 1 + len(m.Tokens)) % len(m.Tokens)
				m.Loading = true
				m.Data = api.AllModelsData{}
				return m, FetchData(m.Tokens[m.ActiveTokenIdx])
			}
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		barWidth := msg.Width - 65
		if barWidth < 20 {
			barWidth = 20
		}
		if barWidth > 50 {
			barWidth = 50
		}
		m.claudeProgress.Width = barWidth
		m.geminiProgress.Width = barWidth

	case NewTokenMsg:
		m.LoggingIn = false
		if msg.Token != nil {
			m.Tokens = append(m.Tokens, msg.Token)
			m.ActiveTokenIdx = len(m.Tokens) - 1
			m.Loading = true
			return m, FetchData(m.Tokens[m.ActiveTokenIdx])
		}
		return m, nil

	case DataMsg:
		m.Data = api.AllModelsData(msg)
		m.Loading = false
		return m, nil

	case ErrMsg:
		m.Err = msg.Err
		m.Loading = false
		m.LoggingIn = false
		return m, nil

	case progress.FrameMsg:
		claudeModel, claudeCmd := m.claudeProgress.Update(msg)
		m.claudeProgress = claudeModel.(progress.Model)

		geminiModel, geminiCmd := m.geminiProgress.Update(msg)
		m.geminiProgress = geminiModel.(progress.Model)

		return m, tea.Batch(claudeCmd, geminiCmd)
	}

	return m, nil
}

func FetchData(tokenFile *config.TokenFile) tea.Cmd {
	return func() tea.Msg {
		if auth.IsExpired(tokenFile.Expired) {
			if err := auth.RefreshToken(tokenFile); err != nil {
				return ErrMsg{fmt.Errorf("token expired and refresh failed: %w", err)}
			}
		}

		data, err := api.CallAPI(tokenFile.AccessToken)
		if err != nil {
			return ErrMsg{err}
		}

		return DataMsg(data)
	}
}

func LoginCmd() tea.Msg {
	token, err := auth.LoginFlow()
	if err != nil {
		return ErrMsg{fmt.Errorf("login failed: %w", err)}
	}
	return NewTokenMsg{Token: token}
}

func (m Model) ClaudeProgress() progress.Model {
	return m.claudeProgress
}

func (m Model) GeminiProgress() progress.Model {
	return m.geminiProgress
}
