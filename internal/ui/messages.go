package ui

import (
	"time"

	"github.com/deLiseLINO/antigravity-quota/internal/api"
	"github.com/deLiseLINO/antigravity-quota/internal/config"
)

type ViewMode int

const (
	ModeGroups ViewMode = iota
	ModeAll
)

type DataMsg struct {
	Key     string
	Data    api.AllModelsData
	Summary api.QuotaSummaryData
}

type ErrMsg struct {
	Key string
	Err error
}

type NewTokenMsg struct{ Token *config.TokenFile }

type AnimationFrameMsg struct {
	Now time.Time
}

type NoticeMsg struct {
	Text string
}

type NoticeTimeoutMsg struct {
	Seq int
}
