package jobs

import (
	"sync"
	"time"
)

type JobStatus struct {
	Instances map[string]InstanceStatus `json:"instances"`
}

type InstanceStatus struct {
	InstanceID string     `json:"instanceId"`
	LastRun    *time.Time `json:"lastRun,omitempty"`
	LastCount  int        `json:"lastCount"`
	LastError  string     `json:"lastError,omitempty"`
	State      string     `json:"state"`
}

type StatusTracker struct {
	mu        sync.RWMutex
	instances map[string]InstanceStatus
}

func NewStatusTracker() *StatusTracker {
	return &StatusTracker{instances: make(map[string]InstanceStatus)}
}

func (s *StatusTracker) Get() JobStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]InstanceStatus, len(s.instances))
	for k, v := range s.instances {
		out[k] = v
	}
	return JobStatus{Instances: out}
}

func (s *StatusTracker) SetLastRun(instanceID string, count int, at time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.instances[instanceID] = InstanceStatus{
		InstanceID: instanceID,
		LastRun:    &at,
		LastCount:  count,
		State:      "idle",
	}
}

func (s *StatusTracker) SetIdle(instanceID string, count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing := s.instances[instanceID]
	existing.State = "idle"
	existing.LastCount = count
	existing.LastError = ""
	s.instances[instanceID] = existing
}

func (s *StatusTracker) SetError(instanceID string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing := s.instances[instanceID]
	existing.State = "error"
	existing.LastError = err.Error()
	s.instances[instanceID] = existing
}

func (s *StatusTracker) RecordManualHunt(instanceID string, count int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	existing := s.instances[instanceID]
	existing.InstanceID = instanceID
	existing.LastRun = &now
	existing.LastCount = count
	existing.State = "idle"
	existing.LastError = ""
	s.instances[instanceID] = existing
}
