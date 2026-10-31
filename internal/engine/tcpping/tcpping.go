package tcpping

import (
	"context"
	"errors"
	"net"
	"strconv"
	"syscall"
	"time"

	"qnetward/internal/engine/report"
)

type Config struct {
	Count      int
	Interval   time.Duration
	Timeout    time.Duration
	Continuous bool
	// AddrCheck, when set, vets every address the host resolves to before the
	// run starts, and again on each connect (guarding against DNS rebinding).
	AddrCheck func(net.IP) error
}

func Run(host string, port int, cfg Config) (report.Report, error) {
	return RunContext(context.Background(), host, port, cfg)
}

func RunContext(ctx context.Context, host string, port int, cfg Config) (report.Report, error) {
	return RunStreamContext(ctx, host, port, cfg, nil)
}

func RunStream(host string, port int, cfg Config, onResult func(report.Result) error) (report.Report, error) {
	return RunStreamContext(context.Background(), host, port, cfg, onResult)
}

func RunStreamContext(ctx context.Context, host string, port int, cfg Config, onResult func(report.Result) error) (report.Report, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if cfg.Count <= 0 && !cfg.Continuous {
		return report.Report{}, errors.New("count must be greater than 0")
	}
	if port <= 0 || port > 65535 {
		return report.Report{}, errors.New("port must be in range 1-65535")
	}
	if cfg.Interval <= 0 {
		cfg.Interval = time.Second
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 2 * time.Second
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	initialCapacity := cfg.Count
	if initialCapacity < 0 {
		initialCapacity = 0
	}
	results := make([]report.Result, 0, initialCapacity)
	dialer := &net.Dialer{Timeout: cfg.Timeout}
	if cfg.AddrCheck != nil {
		if err := checkHost(ctx, host, cfg.AddrCheck); err != nil {
			return report.Report{}, err
		}
		dialer.Control = dialControl(cfg.AddrCheck)
	}
	startedAt := time.Now()

	for i := 1; ; i++ {
		if !cfg.Continuous && i > cfg.Count {
			break
		}
		if err := ctx.Err(); err != nil {
			return buildReport(host, addr, port, results, time.Since(startedAt)), err
		}

		start := time.Now()
		conn, err := dialer.DialContext(ctx, "tcp", addr)
		var res report.Result
		if err != nil {
			if ctx.Err() != nil {
				return buildReport(host, addr, port, results, time.Since(startedAt)), ctx.Err()
			}
			res = report.Result{Seq: i, Error: err.Error()}
		} else {
			rtt := time.Since(start)
			_ = conn.Close()
			res = report.Result{Seq: i, RTT: rtt, Addr: addr}
		}
		results = append(results, res)
		if onResult != nil {
			if err := onResult(res); err != nil {
				return buildReport(host, addr, port, results, time.Since(startedAt)), err
			}
		}

		if cfg.Continuous || i < cfg.Count {
			if err := sleepContext(ctx, cfg.Interval); err != nil {
				return buildReport(host, addr, port, results, time.Since(startedAt)), err
			}
		}
	}

	return buildReport(host, addr, port, results, time.Since(startedAt)), nil
}

func buildReport(host, addr string, port int, results []report.Result, duration time.Duration) report.Report {
	summary := report.Summarize(results, len(results))
	summary.Duration = duration
	return report.Report{
		Protocol: "tcp",
		Target:   host,
		Addr:     addr,
		Port:     port,
		Results:  results,
		Summary:  summary,
	}
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func checkHost(ctx context.Context, host string, check func(net.IP) error) error {
	if ip := net.ParseIP(host); ip != nil {
		return check(ip)
	}
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return err
	}
	for _, addr := range addrs {
		if err := check(addr.IP); err != nil {
			return err
		}
	}
	return nil
}

func dialControl(check func(net.IP) error) func(network, address string, c syscall.RawConn) error {
	return func(_, address string, _ syscall.RawConn) error {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			host = address
		}
		return check(net.ParseIP(host))
	}
}
