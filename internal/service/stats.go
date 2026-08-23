package service

import (
	"task210-clocktrace/internal/model"
)

// Stats 系统统计。
type Stats struct {
	Snapshots      int            `json:"snapshots"`
	Nodes          int            `json:"nodes"`
	Links          int            `json:"links"`
	Samples        int            `json:"samples"`
	AnomalySamples int            `json:"anomaly_samples"`
	Events         int            `json:"events"`
	EventsByStatus map[string]int `json:"events_by_status"`
	Candidates     int            `json:"candidates"`
	Confirmed      int            `json:"confirmed_candidates"`
	SealedEvents   int            `json:"sealed_events"`
}

// ComputeStats 汇总全局统计。
func (a *App) ComputeStats() (*Stats, error) {
	s := &Stats{EventsByStatus: map[string]int{}}

	snaps, err := a.DB.ListSnapshots()
	if err != nil {
		return nil, err
	}
	s.Snapshots = len(snaps)

	for _, snap := range snaps {
		nodes, err := a.DB.ListNodes(snap.ID)
		if err != nil {
			return nil, err
		}
		links, err := a.DB.ListLinks(snap.ID)
		if err != nil {
			return nil, err
		}
		samples, err := a.DB.ListSamplesBySnapshot(snap.ID)
		if err != nil {
			return nil, err
		}
		s.Nodes += len(nodes)
		s.Links += len(links)
		s.Samples += len(samples)
		for _, smp := range samples {
			if smp.Status == model.SampleAnomaly {
				s.AnomalySamples++
			}
		}
	}

	events, err := a.DB.ListAllEvents()
	if err != nil {
		return nil, err
	}
	s.Events = len(events)
	for _, ev := range events {
		s.EventsByStatus[ev.Status]++
		if ev.Status == model.EventSealed {
			s.SealedEvents++
		}
		cands, err := a.DB.ListCandidatesByEvent(ev.ID)
		if err != nil {
			return nil, err
		}
		s.Candidates += len(cands)
		for _, c := range cands {
			if c.Status == model.CandidateConfirmed {
				s.Confirmed++
			}
		}
	}
	return s, nil
}
