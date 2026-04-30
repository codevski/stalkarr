package jobs

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"sleeparr/internal/arr"
	"sleeparr/internal/config"
)

type AgentJob struct {
	cfg      func() config.Config
	cooldown *CooldownTracker
	status   *StatusTracker

	mu      sync.Mutex
	cancel  context.CancelFunc
	running bool
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
	h.spawn(ctx, true)
}

func (h *AgentJob) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.running {
		return
	}
	h.cancel()
	h.running = false
	log.Println("[sleeparr] workers stopped")
}

func (h *AgentJob) Reload(ctx context.Context) {
	h.Stop()
	h.spawn(ctx, false)
}

func (h *AgentJob) spawn(ctx context.Context, runImmediately bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.running {
		return
	}

	cfg := h.cfg()
	if !cfg.Agent.Enabled {
		log.Println("[sleeparr] disabled — set agent.enabled=true to activate")
		return
	}

	cooldownDur := time.Duration(cfg.Agent.CooldownHours) * time.Hour
	if cooldownDur == 0 {
		cooldownDur = 24 * time.Hour
	}
	h.cooldown.SetDuration(cooldownDur)

	runCtx, cancel := context.WithCancel(ctx)
	h.cancel = cancel
	h.running = true

	log.Printf("[sleeparr] starting — interval=%dm, count=%d, cooldown=%dh",
		cfg.Agent.IntervalMinutes, cfg.Agent.EpisodesPerRun, cfg.Agent.CooldownHours)

	go h.sonarrWorker(runCtx, runImmediately)
	go h.maintenanceWorker(runCtx)
}

func (h *AgentJob) sonarrWorker(ctx context.Context, runImmediately bool) {
	interval := time.Duration(h.cfg().Agent.IntervalMinutes) * time.Minute
	if interval == 0 {
		interval = 60 * time.Minute
	}

	if runImmediately {
		h.runSonarrFanOut(ctx)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.runSonarrFanOut(ctx)
		}
	}
}

func (h *AgentJob) runSonarrFanOut(ctx context.Context) {
	instances := h.cfg().Sonarr
	if len(instances) == 0 {
		return
	}

	var wg sync.WaitGroup
	for _, instance := range instances {
		if ctx.Err() != nil {
			return
		}
		wg.Add(1)
		go func(inst config.SonarrInstance) {
			defer wg.Done()
			if err := h.runSonarrInstance(ctx, inst); err != nil {
				log.Printf("[sleeparr] sonarr/%s error: %v", inst.ID, err)
				h.status.SetError(inst.ID, err)
			}
		}(instance)
	}
	wg.Wait()
}

func (h *AgentJob) runSonarrInstance(ctx context.Context, instance config.SonarrInstance) error {
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

func (h *AgentJob) maintenanceWorker(ctx context.Context) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.cooldown.Prune()
		}
	}
}

func (h *AgentJob) RecordManualRun(instanceID string, count int) {
	h.status.RecordManualRun(instanceID, count)
}
