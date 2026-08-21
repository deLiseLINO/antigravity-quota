package ui

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/deLiseLINO/antigravity-quota/internal/api"
	"github.com/deLiseLINO/antigravity-quota/internal/auth"
	"github.com/deLiseLINO/antigravity-quota/internal/config"
)

type Model struct {
	Data             api.AllModelsData
	Summary          api.QuotaSummaryData
	Loading          bool
	DeleteConfirm    bool
	LoggingIn        bool
	Err              error
	Width            int
	Height           int
	Mode             ViewMode
	Tokens           []*config.TokenFile
	ActiveTokenIdx   int
	DataCache        map[string]api.AllModelsData
	SummaryCache     map[string]api.QuotaSummaryData
	LoadingMap       map[string]bool
	ErrorsMap        map[string]error
	barWidth         int
	HelpVisible      bool
	Notice           string
	noticeSeq        int
	barAnimations    map[string]barAnimation
	animationTicking bool
}

func InitialModel(tokens []*config.TokenFile) Model {
	return Model{
		Loading:        true,
		Mode:           ModeGroups,
		Tokens:         tokens,
		ActiveTokenIdx: 0,
		DataCache:      make(map[string]api.AllModelsData),
		SummaryCache:   make(map[string]api.QuotaSummaryData),
		LoadingMap:     make(map[string]bool),
		ErrorsMap:      make(map[string]error),
		barWidth:       40,
		barAnimations:  make(map[string]barAnimation),
	}
}

func (m Model) Init() tea.Cmd {
	if len(m.Tokens) == 0 {
		return nil
	}
	return tea.Batch(FetchData(m.Tokens[m.ActiveTokenIdx]), m.fetchNextCmd())
}

func (m *Model) fetchNextCmd() tea.Cmd {
	if m.DataCache == nil {
		m.DataCache = make(map[string]api.AllModelsData)
	}
	if m.SummaryCache == nil {
		m.SummaryCache = make(map[string]api.QuotaSummaryData)
	}
	if m.LoadingMap == nil {
		m.LoadingMap = make(map[string]bool)
	}
	if m.ErrorsMap == nil {
		m.ErrorsMap = make(map[string]error)
	}

	const maxConcurrentLoads = 3
	currentlyLoading := 0
	for _, isLoading := range m.LoadingMap {
		if isLoading {
			currentlyLoading++
		}
	}
	if currentlyLoading >= maxConcurrentLoads {
		return nil
	}
	availableSlots := maxConcurrentLoads - currentlyLoading

	cmds := make([]tea.Cmd, 0, availableSlots)
	for i, token := range m.Tokens {
		if token == nil || i == m.ActiveTokenIdx {
			continue // active account is fetched separately
		}
		key := tokenKey(token)
		if key == "" {
			continue
		}
		if m.LoadingMap[key] {
			continue
		}
		if _, hasData := m.DataCache[key]; hasData {
			continue
		}
		if _, hasErr := m.ErrorsMap[key]; hasErr {
			continue
		}
		m.LoadingMap[key] = true
		cmds = append(cmds, FetchData(token))
		availableSlots--
		if availableSlots == 0 {
			break
		}
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		rawKey := msg.String()
		keyStr := normalizeHelpKey(rawKey, normalizeKey(rawKey))

		if m.HelpVisible {
			switch keyStr {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "esc", "help":
				m.HelpVisible = false
				return m, nil
			}
			return m, nil
		}

		if m.DeleteConfirm {
			switch keyStr {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "esc":
				m.DeleteConfirm = false
				return m, nil
			case "x", "delete":
				return m.deleteActiveAccount()
			}
			return m, nil
		}

		if m.Err != nil {
			switch keyStr {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "enter", "esc":
				m.Err = nil
				return m, nil
			}
			return m, nil
		}

		switch keyStr {
		case "x", "delete":
			if len(m.Tokens) > 0 {
				m.DeleteConfirm = true
				return m, nil
			}
			return m, nil
		case "esc":
			if m.Notice != "" {
				m.Notice = ""
				return m, nil
			}
			return m, tea.Quit
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			if len(m.Tokens) == 0 {
				return m, nil
			}
			key := tokenKey(m.Tokens[m.ActiveTokenIdx])
			if _, ok := m.DataCache[key]; !ok {
				m.Loading = true
			}
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
		case "help":
			m.HelpVisible = true
			return m, nil
		case "right", "l":
			if len(m.Tokens) > 1 {
				m.ActiveTokenIdx = (m.ActiveTokenIdx + 1) % len(m.Tokens)
				return m.switchToActive()
			}
		case "left", "h":
			if len(m.Tokens) > 1 {
				m.ActiveTokenIdx = (m.ActiveTokenIdx - 1 + len(m.Tokens)) % len(m.Tokens)
				return m.switchToActive()
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
		m.barWidth = barWidth

	case NewTokenMsg:
		m.LoggingIn = false
		if msg.Token != nil {
			m.Tokens = append(m.Tokens, msg.Token)
			m.ActiveTokenIdx = len(m.Tokens) - 1
			m.Loading = true
			m.noticeSeq++
			m.Notice = "Account added"
			return m, tea.Batch(FetchData(m.Tokens[m.ActiveTokenIdx]), scheduleNoticeClearCmd(m.noticeSeq))
		}
		return m, nil

	case DataMsg:
		prevSummary, hadPrevSummary := m.SummaryCache[msg.Key]
		prevData, hadPrevData := m.DataCache[msg.Key]
		wasLoading := m.LoadingMap[msg.Key]

		m.DataCache[msg.Key] = msg.Data
		m.SummaryCache[msg.Key] = msg.Summary
		m.LoadingMap[msg.Key] = false
		delete(m.ErrorsMap, msg.Key)

		if m.ActiveTokenIdx >= 0 && m.ActiveTokenIdx < len(m.Tokens) && msg.Key == tokenKey(m.Tokens[m.ActiveTokenIdx]) {
			m.Data = msg.Data
			m.Summary = msg.Summary
			m.Loading = false
			m.Err = nil
			m.startBucketAnimations(msg.Key, prevSummary, hadPrevSummary, msg.Summary, wasLoading)
			m.startModelAnimations(msg.Key, prevData, hadPrevData, msg.Data, wasLoading)
		}
		return m, tea.Batch(m.fetchNextCmd(), m.ensureAnimationTickCmd())

	case ErrMsg:
		if msg.Key != "" {
			m.ErrorsMap[msg.Key] = msg.Err
			m.LoadingMap[msg.Key] = false
		}
		if m.ActiveTokenIdx >= 0 && m.ActiveTokenIdx < len(m.Tokens) && msg.Key == tokenKey(m.Tokens[m.ActiveTokenIdx]) {
			if _, ok := m.DataCache[msg.Key]; !ok {
				m.Err = msg.Err
				m.Loading = false
			}
		} else if msg.Key == "" {
			m.Err = msg.Err
			m.Loading = false
		}
		m.LoggingIn = false
		return m, nil

	case AnimationFrameMsg:
		if !m.advanceBarAnimations(msg.Now) {
			m.animationTicking = false
			return m, nil
		}
		return m, animationTickCmd()

	case NoticeTimeoutMsg:
		if msg.Seq == m.noticeSeq {
			m.Notice = ""
		}
		return m, nil
	}

	return m, nil
}

func (m Model) switchToActive() (tea.Model, tea.Cmd) {
	key := tokenKey(m.Tokens[m.ActiveTokenIdx])
	if data, ok := m.DataCache[key]; ok {
		m.Data = data
		m.Summary = m.SummaryCache[key]
		m.Loading = false
		m.Err = nil
		m.startBucketAnimationsFromZero(key, m.Summary)
		m.startModelAnimationsFromZero(key, m.Data)
		return m, tea.Batch(FetchData(m.Tokens[m.ActiveTokenIdx]), m.ensureAnimationTickCmd())
	}
	m.Loading = true
	m.Err = nil
	return m, FetchData(m.Tokens[m.ActiveTokenIdx])
}

func (m Model) deleteActiveAccount() (tea.Model, tea.Cmd) {
	token := m.Tokens[m.ActiveTokenIdx]
	key := tokenKey(token)
	if err := os.Remove(token.FilePath); err == nil {
		m.Tokens = append(m.Tokens[:m.ActiveTokenIdx], m.Tokens[m.ActiveTokenIdx+1:]...)
		delete(m.DataCache, key)
		delete(m.SummaryCache, key)
		delete(m.LoadingMap, key)
		delete(m.ErrorsMap, key)
		m.removeAnimationsForAccount(key)
		m.pruneBarAnimations()
		m.DeleteConfirm = false
		if len(m.Tokens) == 0 {
			m.ActiveTokenIdx = 0
			m.Data = api.AllModelsData{}
			m.Summary = api.QuotaSummaryData{}
			return m, nil
		}
		if m.ActiveTokenIdx >= len(m.Tokens) {
			m.ActiveTokenIdx = len(m.Tokens) - 1
		}
		return m.switchToActive()
	}
	m.DeleteConfirm = false
	if len(m.Tokens) > 0 {
		if m.ActiveTokenIdx >= len(m.Tokens) {
			m.ActiveTokenIdx = len(m.Tokens) - 1
		}
		return m.switchToActive()
	}
	return m, nil
}

func scheduleNoticeClearCmd(seq int) tea.Cmd {
	return tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
		return NoticeTimeoutMsg{Seq: seq}
	})
}

func tokenKey(t *config.TokenFile) string {
	if t == nil {
		return ""
	}
	return t.FilePath
}

func FetchData(tokenFile *config.TokenFile) tea.Cmd {
	key := tokenKey(tokenFile)
	return func() tea.Msg {
		if auth.IsExpired(tokenFile.Expired) {
			if err := auth.RefreshToken(tokenFile); err != nil {
				return ErrMsg{Key: key, Err: fmt.Errorf("token expired and refresh failed: %w", err)}
			}
		}

		summary, err := api.CallQuotaSummary(tokenFile.AccessToken)
		if err != nil {
			return ErrMsg{Key: key, Err: err}
		}

		data, err := api.CallAPI(tokenFile.AccessToken)
		if err != nil {
			return ErrMsg{Key: key, Err: err}
		}

		return DataMsg{Key: key, Data: data, Summary: summary}
	}
}

func LoginCmd() tea.Msg {
	token, err := auth.LoginFlow()
	if err != nil {
		return ErrMsg{Key: "", Err: fmt.Errorf("login failed: %w", err)}
	}
	return NewTokenMsg{Token: token}
}
