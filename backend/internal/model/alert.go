package model

import "time"

type Alert struct {
	Resource   string  `json:"resource"`
	Value      float64 `json:"value"`
	Threshold  float64 `json:"threshold"`
	Message    string  `json:"message"`
	IsCritical bool    `json:"is_critical"`
}

type AlertStatus struct {
	Triggered  bool      `json:"triggered"`
	Alerts     []Alert   `json:"alerts"`
	EvaluatedAt time.Time `json:"evaluated_at"`
}

type MetricsStreamEvent struct {
	Snapshot MetricSnapshot `json:"snapshot"`
	Alerts   AlertStatus    `json:"alerts"`
}
