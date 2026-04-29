package jobs

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"sleeparr/internal/arr"
	"sleeparr/internal/config"
)

type AgentJob struct {
	cfg      func() config.Config
	cooldown *CooldownTracker
	status   *StatusTracker
}

func NewAgentJob(cfg func() config.Config) *AgentJob {
	cooldownDur := time.Duration(cfg().Agent.CooldownHours) * time.Hour
	if cooldownDur == 0 {
		cooldownDur = 24 * time.Hour
	}
	return &AgentJob{
		cfg:      cfg,
		cooldown: NewCooldownTracker(cooldownDur),
		status:   NewStatusTracker(),
	}
}

func (h *AgentJob) Status() JobStatus {
	return h.status.Get()
}

func (h *AgentJob) Start(ctx context.Context) {
	cfg := h.cfg()
	if !cfg.Agent.Enabled {
		log.Println("[sleeparr] disabled — set agent.enabled=true in config to activate")
		return
	}
	log.Printf("[sleeparr] starting — interval=%dm, count=%d, cooldown=%dh",
		h.cfg().Agent.IntervalMinutes, h.cfg().Agent.EpisodesPerRun, h.cfg().Agent.CooldownHours)

	// Run immediately on startup, then on the ticker.
	h.runAll(ctx)

	interval := time.Duration(h.cfg().Agent.IntervalMinutes) * time.Minute
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
			log.Println("[sleeparr] shutting down")
			return
		case <-ticker.C:
			h.runAll(ctx)
		case <-pruneTicker.C:
			h.cooldown.Prune()
		}
	}
}

func (h *AgentJob) runAll(ctx context.Context) {
	for _, instance := range h.cfg().Sonarr {
		if ctx.Err() != nil {
			return
		}
		if err := h.runInstance(ctx, instance); err != nil {
			log.Printf("[sleeparr] sonarr/%s error: %v", instance.ID, err)
			h.status.SetError(instance.ID, err)
		}
	}
}

func (h *AgentJob) runInstance(ctx context.Context, instance config.SonarrInstance) error {
	client := arr.NewSonarrClient(instance.URL, instance.APIKey)

	result, err := client.GetMissingEpisodes(1, 500, "")
	if err != nil {
		return fmt.Errorf("fetch missing: %w", err)
	}

	if len(result.Episodes) == 0 {
		log.Printf("[sleeparr] sonarr/%s — no missing episodes, nothing to do", instance.ID)
		h.status.SetIdle(instance.ID, 0)
		return nil
	}

	var eligible []arr.Episode
	for _, ep := range result.Episodes {
		if h.cooldown.IsReady(instance.ID, ep.ID) {
			eligible = append(eligible, ep)
		}
	}

	if len(eligible) == 0 {
		log.Printf("[sleeparr] sonarr/%s — %d missing but all on cooldown", instance.ID, len(result.Episodes))
		h.status.SetIdle(instance.ID, 0)
		return nil
	}

	rand.Shuffle(len(eligible), func(i, j int) { eligible[i], eligible[j] = eligible[j], eligible[i] })
	count := h.cfg().Agent.EpisodesPerRun
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

	log.Printf("[sleeparr] sonarr/%s — finding %d episode(s) (%d eligible, %d total missing)",
		instance.ID, len(ids), len(eligible), len(result.Episodes))

	runResult, err := client.TriggerEpisodeSearch(ids)
	if err != nil {
		return fmt.Errorf("trigger search: %w", err)
	}

	for _, ep := range chosen {
		h.cooldown.MarkRun(instance.ID, ep.ID)
	}

	log.Printf("[sleeparr] sonarr/%s — %s (command ID: %d)", instance.ID, runResult.Message, runResult.CommandID)
	h.status.SetLastRun(instance.ID, len(ids), time.Now())
	return nil
}

func (h *AgentJob) RecordManualRun(instanceID string, count int) {
	h.status.RecordManualRun(instanceID, count)
}
