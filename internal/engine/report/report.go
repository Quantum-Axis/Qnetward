package report

import (
	"math"
	"sort"
	"time"
)

type Result struct {
	Seq   int           `json:"seq"`
	RTT   time.Duration `json:"rtt,omitempty"`
	Error string        `json:"error,omitempty"`
	Addr  string        `json:"addr,omitempty"`
}

type Summary struct {
	Sent       int           `json:"sent"`
	Recv       int           `json:"recv"`
	Loss       float64       `json:"loss"`
	Min        time.Duration `json:"min,omitempty"`
	Avg        time.Duration `json:"avg,omitempty"`
	Max        time.Duration `json:"max,omitempty"`
	Median     time.Duration `json:"median,omitempty"`
	StdDev     time.Duration `json:"stddev,omitempty"`
	Jitter     time.Duration `json:"jitter,omitempty"`
	LossStreak int           `json:"loss_streak,omitempty"`
	Duration   time.Duration `json:"duration,omitempty"`
}

type Report struct {
	Protocol string   `json:"protocol"`
	Target   string   `json:"target"`
	Addr     string   `json:"addr"`
	Port     int      `json:"port,omitempty"`
	Results  []Result `json:"results"`
	Summary  Summary  `json:"summary"`
}

func Summarize(results []Result, sent int) Summary {
	recv := 0
	rtts := make([]time.Duration, 0, len(results))
	maxLossStreak := 0
	currentLossStreak := 0
	for _, r := range results {
		if r.Error == "" {
			recv++
			rtts = append(rtts, r.RTT)
			currentLossStreak = 0
			continue
		}
		currentLossStreak++
		if currentLossStreak > maxLossStreak {
			maxLossStreak = currentLossStreak
		}
	}

	s := Summary{Sent: sent, Recv: recv, LossStreak: maxLossStreak}
	if sent > 0 {
		s.Loss = (float64(sent-recv) / float64(sent)) * 100
	}
	if recv == 0 {
		return s
	}

	sorted := append([]time.Duration(nil), rtts...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	min := sorted[0]
	max := sorted[0]
	var total float64
	for _, rtt := range sorted {
		if rtt < min {
			min = rtt
		}
		if rtt > max {
			max = rtt
		}
		total += float64(rtt)
	}
	avg := total / float64(recv)

	mid := len(sorted) / 2
	median := sorted[mid]
	if len(sorted)%2 == 0 {
		median = time.Duration((float64(sorted[mid-1]) + float64(sorted[mid])) / 2)
	}

	var variance float64
	for _, rtt := range sorted {
		d := float64(rtt) - avg
		variance += d * d
	}
	stddev := math.Sqrt(variance / float64(recv))

	var jitterTotal float64
	if len(sorted) > 1 {
		for i := 1; i < len(rtts); i++ {
			delta := math.Abs(float64(rtts[i]) - float64(rtts[i-1]))
			jitterTotal += delta
		}
	}
	jitter := 0.0
	if len(rtts) > 1 {
		jitter = jitterTotal / float64(len(rtts)-1)
	}

	s.Min = min
	s.Max = max
	s.Avg = time.Duration(avg)
	s.Median = median
	s.StdDev = time.Duration(stddev)
	s.Jitter = time.Duration(jitter)
	return s
}
