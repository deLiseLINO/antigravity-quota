package ui

import (
	"github.com/deLiseLINO/antigravity-quota/internal/api"
	"github.com/deLiseLINO/antigravity-quota/internal/config"
)

type ViewMode int

const (
	ModeGroups ViewMode = iota
	ModeAll
)

type DataMsg api.AllModelsData

type ErrMsg struct{ Err error }

type NewTokenMsg struct{ Token *config.TokenFile }
