# AGQ (Antigravity Quota Monitor)

A TUI for monitoring Antigravity account quotas written in Go using [Bubble Tea](https://github.com/charmbracelet/bubbletea).

![Demo](demo.gif)

## Features

- Multiple Antigravity accounts simultaneously
- Two display modes: grouped by models or flat list
- OAuth authentication via browser

## Installation

```bash
go install github.com/deLiseLINO/antigravity-quota/cmd/agq@latest
```

Build from source:

```bash
git clone https://github.com/deLiseLINO/antigravity-quota.git
cd antigravity-quota
go install ./cmd/agq
```

## Usage

Run the app:

```bash
agq
```

On first launch press `n` to add an account — browser will open for Google OAuth authentication.

## Controls

- `n` — add new account
- `←` / `→` (or `h` / `l`) — switch between accounts
- `r` — refresh data
- `tab` (or `m`) — toggle display mode
- `x` — delete current account
- `q` (or `Esc`) — quit
