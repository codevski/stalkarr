package jobs

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"stalkarr/internal/arr"
	"stalkarr/internal/config"
)

type HunterJob struct {
	cfg      func() config.Config
	cooldown *CooldownTracker
	status   *StatusTracker
}

func NewHunterJob(cfg func() config.Config) *HunterJob {
	cooldownDur := time.Duration(cfg().Hunt.CooldownHours) * time.Hour
	if cooldownDur == 0 {
		cooldownDur = 24 * time.Hour // sensible default
	}
	return &HunterJob{
		cfg:      cfg,
		cooldown: NewCooldownTracker(cooldownDur),
		status:   NewStatusTracker(),
	}
}

// Status returns the current job status for the dashboard/UI.
func (h *HunterJob) Status() JobStatus {
	return h.status.Get()
}

// Start begins the background hunt loop. It blocks until ctx is cancelled,
// so run it in a goroutine. Sends a prune of the cooldown map every 6 hours.
func (h *HunterJob) Start(ctx context.Context) {
	cfg := h.cfg()
	if !cfg.Hunt.Enabled {
		log.Println("[hunter] disabled — set hunt.enabled=true in config to activate")
		return
	}
	log.Printf("[hunter] starting — interval=%dm, count=%d, cooldown=%dh",
		h.cfg().Hunt.IntervalMinutes, h.cfg().Hunt.EpisodesPerRun, h.cfg().Hunt.CooldownHours)

	// Run immediately on startup, then on the ticker.
	h.runAll(ctx)

	interval := time.Duration(h.cfg().Hunt.IntervalMinutes) * time.Minute
	if interval == 0 {
		interval = 60 * time.Minute
	}
	ticker := time.NewTicker(interval)
	pruneTicker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	defer pruneTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[hunter] shutting down")
			return
		case <-ticker.C:
			h.runAll(ctx)
		case <-pruneTicker.C:
			h.cooldown.Prune()
		}
	}
}

// runAll iterates over every configured Sonarr instance and hunts for each.
func (h *HunterJob) runAll(ctx context.Context) {
	for _, instance := range h.cfg().Sonarr {
		if ctx.Err() != nil {
			return
		}
		if err := h.runInstance(ctx, instance); err != nil {
			log.Printf("[hunter] sonarr/%s error: %v", instance.ID, err)
			h.status.SetError(instance.ID, err)
		}
	}
}

// runInstance fetches missing episodes for one Sonarr instance, picks up to
// EpisodesPerRun that aren't on cooldown (random selection), and hunts them.
func (h *HunterJob) runInstance(ctx context.Context, instance config.SonarrInstance) error {
	client := arr.NewSonarrClient(instance.URL, instance.APIKey)

	// Fetch a large page so we have a good pool to randomly pick from.
	// Sonarr's missing list is already sorted by airdate desc, so grabbing
	// page 1 / 500 gives us the most recently aired missing episodes.
	result, err := client.GetMissingEpisodes(1, 500, "")
	if err != nil {
		return fmt.Errorf("fetch missing: %w", err)
	}

	if len(result.Episodes) == 0 {
		log.Printf("[hunter] sonarr/%s — no missing episodes, nothing to do", instance.ID)
		h.status.SetIdle(instance.ID, 0)
		return nil
	}

	// Filter to episodes that are off cooldown.
	var eligible []arr.Episode
	for _, ep := range result.Episodes {
		if h.cooldown.IsReady(instance.ID, ep.ID) {
			eligible = append(eligible, ep)
		}
	}

	if len(eligible) == 0 {
		log.Printf("[hunter] sonarr/%s — %d missing but all on cooldown", instance.ID, len(result.Episodes))
		h.status.SetIdle(instance.ID, 0)
		return nil
	}

	// Shuffle and take up to EpisodesPerRun.
	rand.Shuffle(len(eligible), func(i, j int) { eligible[i], eligible[j] = eligible[j], eligible[i] })
	count := h.cfg().Hunt.EpisodesPerRun
	if count <= 0 {
		count = 10
	}
	if count > len(eligible) {
		count = len(eligible)
	}
	chosen := eligible[:count]

	ids := make([]int, len(chosen))
	for i, ep := range chosen {
		ids[i] = ep.ID
	}

	log.Printf("[hunter] sonarr/%s — hunting %d episode(s) (%d eligible, %d total missing)",
		instance.ID, len(ids), len(eligible), len(result.Episodes))

	huntResult, err := client.TriggerEpisodeSearch(ids)
	if err != nil {
		return fmt.Errorf("trigger search: %w", err)
	}

	// Mark all as hunted so they go on cooldown.
	for _, ep := range chosen {
		h.cooldown.MarkHunted(instance.ID, ep.ID)
	}

	log.Printf("[hunter] sonarr/%s — %s (command ID: %d)", instance.ID, huntResult.Message, huntResult.CommandID)
	h.status.SetLastRun(instance.ID, len(ids), time.Now())
	return nil
}

func (h *HunterJob) RecordManualHunt(instanceID string, count int) {
	h.status.RecordManualHunt(instanceID, count)
}
