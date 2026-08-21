package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/deLiseLINO/antigravity-quota/internal/api"
)

type barAnimation struct {
	From      float64
	To        float64
	Current   float64
	StartedAt time.Time
	Duration  time.Duration
}

const (
	animationFrameInterval   = 16 * time.Millisecond
	unifiedAnimationDuration = 1000 * time.Millisecond
)

func (m *Model) ensureAnimationTickCmd() tea.Cmd {
	if !m.hasActiveAnimations() {
		m.animationTicking = false
		return nil
	}
	if m.animationTicking {
		return nil
	}
	m.animationTicking = true
	return animationTickCmd()
}

func animationTickCmd() tea.Cmd {
	return tea.Tick(animationFrameInterval, func(now time.Time) tea.Msg {
		return AnimationFrameMsg{Now: now}
	})
}

func bucketAnimationKey(accountKey string, b api.QuotaBucket) string {
	id := b.BucketID
	if id == "" {
		id = b.DisplayName
	}
	if id == "" {
		id = b.Window
	}
	return fmt.Sprintf("%s|b|%s", accountKey, id)
}

func modelAnimationKey(accountKey string, mq api.ModelQuota) string {
	return fmt.Sprintf("%s|m|%s", accountKey, mq.Name)
}

func (m *Model) startBucketAnimations(accountKey string, prev api.QuotaSummaryData, hadPrev bool, next api.QuotaSummaryData, wasLoading bool) {
	if accountKey == "" {
		return
	}
	m.removeBucketAnimationsForAccount(accountKey)

	previousByKey := make(map[string]float64)
	for _, group := range prev.Groups {
		for _, bucket := range group.Buckets {
			previousByKey[bucketAnimationKey(accountKey, bucket)] = clampRatio(bucket.RemainingFraction)
		}
	}

	for _, group := range next.Groups {
		for _, bucket := range group.Buckets {
			key := bucketAnimationKey(accountKey, bucket)
			target := clampRatio(bucket.RemainingFraction)
			from := 0.0
			if hadPrev && !wasLoading {
				if prevRatio, ok := previousByKey[key]; ok {
					from = prevRatio
				}
			}
			if from == target {
				delete(m.barAnimations, key)
				continue
			}
			if m.barAnimations == nil {
				m.barAnimations = make(map[string]barAnimation)
			}
			m.barAnimations[key] = barAnimation{
				From:      from,
				To:        target,
				Current:   from,
				StartedAt: time.Now(),
				Duration:  unifiedAnimationDuration,
			}
		}
	}
}

func (m *Model) startModelAnimations(accountKey string, prev api.AllModelsData, hadPrev bool, next api.AllModelsData, wasLoading bool) {
	if accountKey == "" {
		return
	}
	m.removeModelAnimationsForAccount(accountKey)

	previousByKey := make(map[string]float64)
	for _, mq := range prev.All {
		previousByKey[modelAnimationKey(accountKey, mq)] = clampRatio(mq.Quota)
	}

	for _, mq := range next.All {
		key := modelAnimationKey(accountKey, mq)
		target := clampRatio(mq.Quota)
		from := 0.0
		if hadPrev && !wasLoading {
			if prevRatio, ok := previousByKey[key]; ok {
				from = prevRatio
			}
		}
		if from == target {
			delete(m.barAnimations, key)
			continue
		}
		if m.barAnimations == nil {
			m.barAnimations = make(map[string]barAnimation)
		}
		m.barAnimations[key] = barAnimation{
			From:      from,
			To:        target,
			Current:   from,
			StartedAt: time.Now(),
			Duration:  unifiedAnimationDuration,
		}
	}
}

func (m *Model) startBucketAnimationsFromZero(accountKey string, next api.QuotaSummaryData) {
	if accountKey == "" {
		return
	}
	m.removeBucketAnimationsForAccount(accountKey)
	for _, group := range next.Groups {
		for _, bucket := range group.Buckets {
			key := bucketAnimationKey(accountKey, bucket)
			target := clampRatio(bucket.RemainingFraction)
			if target == 0 {
				delete(m.barAnimations, key)
				continue
			}
			if m.barAnimations == nil {
				m.barAnimations = make(map[string]barAnimation)
			}
			m.barAnimations[key] = barAnimation{
				From:      0,
				To:        target,
				Current:   0,
				StartedAt: time.Now(),
				Duration:  unifiedAnimationDuration,
			}
		}
	}
}

func (m *Model) startModelAnimationsFromZero(accountKey string, next api.AllModelsData) {
	if accountKey == "" {
		return
	}
	m.removeModelAnimationsForAccount(accountKey)
	for _, mq := range next.All {
		key := modelAnimationKey(accountKey, mq)
		target := clampRatio(mq.Quota)
		if target == 0 {
			delete(m.barAnimations, key)
			continue
		}
		if m.barAnimations == nil {
			m.barAnimations = make(map[string]barAnimation)
		}
		m.barAnimations[key] = barAnimation{
			From:      0,
			To:        target,
			Current:   0,
			StartedAt: time.Now(),
			Duration:  unifiedAnimationDuration,
		}
	}
}

func (m *Model) removeAnimationsForAccount(accountKey string) {
	if accountKey == "" || len(m.barAnimations) == 0 {
		return
	}
	prefix := accountKey + "|"
	for key := range m.barAnimations {
		if strings.HasPrefix(key, prefix) {
			delete(m.barAnimations, key)
		}
	}
}

func (m *Model) removeBucketAnimationsForAccount(accountKey string) {
	if accountKey == "" || len(m.barAnimations) == 0 {
		return
	}
	prefix := accountKey + "|b|"
	for key := range m.barAnimations {
		if strings.HasPrefix(key, prefix) {
			delete(m.barAnimations, key)
		}
	}
}

func (m *Model) removeModelAnimationsForAccount(accountKey string) {
	if accountKey == "" || len(m.barAnimations) == 0 {
		return
	}
	prefix := accountKey + "|m|"
	for key := range m.barAnimations {
		if strings.HasPrefix(key, prefix) {
			delete(m.barAnimations, key)
		}
	}
}

func (m *Model) advanceBarAnimations(now time.Time) bool {
	if len(m.barAnimations) == 0 {
		return false
	}
	for key, anim := range m.barAnimations {
		if anim.Duration <= 0 {
			delete(m.barAnimations, key)
			continue
		}
		elapsed := now.Sub(anim.StartedAt)
		if elapsed <= 0 {
			continue
		}
		progress := float64(elapsed) / float64(anim.Duration)
		if progress >= 1 {
			delete(m.barAnimations, key)
			continue
		}
		if progress < 0 {
			progress = 0
		}
		eased := 1 - (1-progress)*(1-progress)
		anim.Current = anim.From + (anim.To-anim.From)*eased
		m.barAnimations[key] = anim
	}
	return len(m.barAnimations) > 0
}

func (m *Model) hasActiveAnimations() bool {
	return len(m.barAnimations) > 0
}

func (m Model) barRatio(key string, fallback float64) float64 {
	if key == "" || len(m.barAnimations) == 0 {
		return fallback
	}
	anim, ok := m.barAnimations[key]
	if !ok {
		return fallback
	}
	return anim.Current
}

func (m *Model) pruneBarAnimations() {
	if len(m.barAnimations) == 0 {
		return
	}
	valid := make(map[string]struct{}, len(m.Tokens))
	for _, token := range m.Tokens {
		if token == nil {
			continue
		}
		if key := tokenKey(token); key != "" {
			valid[key] = struct{}{}
		}
	}
	for key := range m.barAnimations {
		accountKey := key
		if idx := strings.Index(key, "|"); idx >= 0 {
			accountKey = key[:idx]
		}
		if _, ok := valid[accountKey]; !ok {
			delete(m.barAnimations, key)
		}
	}
}
