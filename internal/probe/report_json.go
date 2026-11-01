package probe

import (
	"fmt"
	"time"

	"qnetward/internal/engine/portscan"
	"qnetward/internal/engine/report"
)

type JSONResult struct {
	Seq   int      `json:"seq,omitempty"`
	Port  int      `json:"port,omitempty"`
	State string   `json:"state,omitempty"`
	RTT   string   `json:"rtt,omitempty"`
	RTTMs *float64 `json:"rtt_ms,omitempty"`
	Error string   `json:"error,omitempty"`
	Addr  string   `json:"addr,omitempty"`
}

type JSONSummary struct {
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

type JSONReport struct {
	Protocol string       `json:"protocol"`
	Target   string       `json:"target"`
	Addr     string       `json:"addr"`
	Port     int          `json:"port,omitempty"`
	Results  []JSONResult `json:"results"`
	Summary  JSONSummary  `json:"summary"`
}

func ToJSONReport(rep report.Report) JSONReport {
	jr := JSONReport{
		Protocol: rep.Protocol,
		Target:   rep.Target,
		Addr:     rep.Addr,
		Port:     rep.Port,
		Summary:  ToJSONSummary(rep.Summary),
	}

	jr.Results = make([]JSONResult, 0, len(rep.Results))
	for _, r := range rep.Results {
		jr.Results = append(jr.Results, ToJSONResult(r))
	}

	return jr
}

func ToJSONPortScanReport(rep portscan.Report) JSONReport {
	jr := JSONReport{
		Protocol: rep.Protocol,
		Target:   rep.Target,
		Addr:     rep.Addr,
		Summary:  ToJSONPortScanSummary(rep.Summary),
	}

	jr.Results = make([]JSONResult, 0, len(rep.Results))
	for _, r := range rep.Results {
		jr.Results = append(jr.Results, ToJSONPortScanResult(r))
	}

	return jr
}

func ToJSONSummary(s report.Summary) JSONSummary {
	js := JSONSummary{
		Sent:       s.Sent,
		Recv:       s.Recv,
		Loss:       s.Loss,
		LossStreak: s.LossStreak,
	}
	if s.Recv > 0 {
		js.Min = FormatDuration(s.Min)
		js.Avg = FormatDuration(s.Avg)
		js.Max = FormatDuration(s.Max)
		js.Median = FormatDuration(s.Median)
		js.StdDev = FormatDuration(s.StdDev)
		js.Jitter = FormatDuration(s.Jitter)
	}
	if s.Duration > 0 {
		js.Duration = FormatDuration(s.Duration)
	}
	return js
}

func ToJSONPortScanSummary(s portscan.Summary) JSONSummary {
	return JSONSummary{
		Scanned:  s.Scanned,
		Open:     s.Open,
		Closed:   s.Closed,
		Timeout:  s.Timeout,
		Duration: s.Duration.Round(time.Millisecond).String(),
	}
}

func ToJSONResult(r report.Result) JSONResult {
	jr := JSONResult{Seq: r.Seq}
	if r.Error != "" {
		jr.Error = r.Error
	} else {
		jr.RTT = FormatDuration(r.RTT)
		ms := float64(r.RTT) / float64(time.Millisecond)
		jr.RTTMs = &ms
	}
	if r.Addr != "" {
		jr.Addr = r.Addr
	}
	return jr
}

func ToJSONPortScanResult(r portscan.Result) JSONResult {
	jr := JSONResult{
		Port:  r.Port,
		State: r.State,
		Error: r.Error,
		Addr:  r.Addr,
	}
	if r.State == "open" {
		jr.RTT = FormatDuration(r.RTT)
		ms := float64(r.RTT) / float64(time.Millisecond)
		jr.RTTMs = &ms
	}
	return jr
}

// FormatDuration renders a duration in milliseconds with microsecond precision, as reports show it.
func FormatDuration(d time.Duration) string {
	ms := float64(d) / float64(time.Millisecond)
	return fmt.Sprintf("%.3fms", ms)
}
