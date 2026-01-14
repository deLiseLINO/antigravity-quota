package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/deLiseLINO/antigravity-quota/internal/auth"
	"github.com/deLiseLINO/antigravity-quota/internal/config"
	"github.com/deLiseLINO/antigravity-quota/internal/ui"
)

func main() {
	tokens, err := config.LoadAllTokens()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading tokens: %v\n", err)
	}

	if len(tokens) == 0 {
		fmt.Println("No accounts found. Starting login flow...")
		token, err := auth.LoginFlow()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Login failed: %v\n", err)
			os.Exit(1)
		}
		tokens = append(tokens, token)
	}

	p := tea.NewProgram(ui.InitialModel(tokens), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
