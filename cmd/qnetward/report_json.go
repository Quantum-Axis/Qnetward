package main

import (
	"time"

	"qnetward/internal/engine/portscan"
	"qnetward/internal/engine/report"
)

type jsonResult struct {
	Seq   int      `json:"seq,omitempty"`
	Port  int      `json:"port,omitempty"`
	State string   `json:"state,omitempty"`
	RTT   string   `json:"rtt,omitempty"`
	RTTMs *float64 `json:"rtt_ms,omitempty"`
	Error string   `json:"error,omitempty"`
	Addr  string   `json:"addr,omitempty"`
}

type jsonSummary struct {
	Sent       int     `json:"sent"`
	Recv       int     `json:"recv"`
	Loss       float64 `json:"loss"`
	Min        string  `json:"min,omitempty"`
	Avg        string  `json:"avg,omitempty"`
	Max        string  `json:"max,omitempty"`
	Median     string  `json:"median,omitempty"`
	StdDev     string  `json:"stddev,omitempty"`
	Jitter     string  `json:"jitter,omitempty"`
	LossStreak int     `json:"loss_streak,omitempty"`
	Scanned    int     `json:"scanned,omitempty"`
	Open       int     `json:"open,omitempty"`
	Closed     int     `json:"closed,omitempty"`
	Timeout    int     `json:"timeout,omitempty"`
	Duration   string  `json:"duration,omitempty"`
}

type jsonReport struct {
	Protocol string       `json:"protocol"`
	Target   string       `json:"target"`
	Addr     string       `json:"addr"`
	Port     int          `json:"port,omitempty"`
	Results  []jsonResult `json:"results"`
	Summary  jsonSummary  `json:"summary"`
}

func toJSONReport(rep report.Report) jsonReport {
	jr := jsonReport{
		Protocol: rep.Protocol,
		Target:   rep.Target,
		Addr:     rep.Addr,
		Port:     rep.Port,
		Summary:  toJSONSummary(rep.Summary),
	}

	jr.Results = make([]jsonResult, 0, len(rep.Results))
	for _, r := range rep.Results {
		jr.Results = append(jr.Results, toJSONResult(r))
	}

	return jr
}

func toJSONPortScanReport(rep portscan.Report) jsonReport {
	jr := jsonReport{
		Protocol: rep.Protocol,
		Target:   rep.Target,
		Addr:     rep.Addr,
		Summary:  toJSONPortScanSummary(rep.Summary),
	}

	jr.Results = make([]jsonResult, 0, len(rep.Results))
	for _, r := range rep.Results {
		jr.Results = append(jr.Results, toJSONPortScanResult(r))
	}

	return jr
}

func toJSONSummary(s report.Summary) jsonSummary {
	js := jsonSummary{
		Sent:       s.Sent,
		Recv:       s.Recv,
		Loss:       s.Loss,
		LossStreak: s.LossStreak,
	}
	if s.Recv > 0 {
		js.Min = formatDuration(s.Min)
		js.Avg = formatDuration(s.Avg)
		js.Max = formatDuration(s.Max)
		js.Median = formatDuration(s.Median)
		js.StdDev = formatDuration(s.StdDev)
		js.Jitter = formatDuration(s.Jitter)
	}
	if s.Duration > 0 {
		js.Duration = formatDuration(s.Duration)
	}
	return js
}

func toJSONPortScanSummary(s portscan.Summary) jsonSummary {
	return jsonSummary{
		Scanned:  s.Scanned,
		Open:     s.Open,
		Closed:   s.Closed,
		Timeout:  s.Timeout,
		Duration: s.Duration.Round(time.Millisecond).String(),
	}
}

func toJSONResult(r report.Result) jsonResult {
	jr := jsonResult{Seq: r.Seq}
	if r.Error != "" {
		jr.Error = r.Error
	} else {
		jr.RTT = formatDuration(r.RTT)
		ms := float64(r.RTT) / float64(time.Millisecond)
		jr.RTTMs = &ms
	}
	if r.Addr != "" {
		jr.Addr = r.Addr
	}
	return jr
}

func toJSONPortScanResult(r portscan.Result) jsonResult {
	jr := jsonResult{
		Port:  r.Port,
		State: r.State,
		Error: r.Error,
		Addr:  r.Addr,
	}
	if r.State == "open" {
		jr.RTT = formatDuration(r.RTT)
		ms := float64(r.RTT) / float64(time.Millisecond)
		jr.RTTMs = &ms
	}
	return jr
}
