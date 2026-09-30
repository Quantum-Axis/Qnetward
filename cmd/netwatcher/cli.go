package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"netwatcher/internal/dnslookup"
	"netwatcher/internal/ping"
	"netwatcher/internal/portscan"
	"netwatcher/internal/report"
	"netwatcher/internal/tcpping"
)

func runPing(args []string) {
	fs := flag.NewFlagSet("ping", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	count := fs.Int("count", 4, "number of probes")
	interval := fs.Duration("interval", time.Second, "interval between probes")
	timeout := fs.Duration("timeout", 2*time.Second, "per-probe timeout")
	size := fs.Int("size", 32, "ICMP payload size in bytes")
	ipv6 := fs.Bool("ipv6", false, "force IPv6")
	jsonOut := fs.Bool("json", false, "JSON output")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: netwatcher ping [flags] <host>")
		fs.PrintDefaults()
	}

	if err := fs.Parse(normalizeArgs(fs, args)); err != nil {
		os.Exit(2)
	}

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(2)
	}

	host := fs.Arg(0)
	cfg, err := normalizePingConfig(*count, *interval, *timeout, *size, *ipv6)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	rep, err := ping.Run(host, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *jsonOut {
		printJSON(rep)
		return
	}
	printPing(rep)
}

func runTCPPing(args []string) {
	fs := flag.NewFlagSet("tcpping", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	count := fs.Int("count", 4, "number of probes")
	interval := fs.Duration("interval", time.Second, "interval between probes")
	timeout := fs.Duration("timeout", 2*time.Second, "per-probe timeout")
	port := fs.Int("port", 80, "TCP port")
	jsonOut := fs.Bool("json", false, "JSON output")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: netwatcher tcpping [flags] <host>[:port]")
		fs.PrintDefaults()
	}

	if err := fs.Parse(normalizeArgs(fs, args)); err != nil {
		os.Exit(2)
	}

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(2)
	}

	host := fs.Arg(0)
	portFromFlag := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "port" {
			portFromFlag = true
		}
	})

	if !portFromFlag {
		parsedHost, parsedPort, ok := splitHostPort(host)
		if ok {
			host = parsedHost
			*port = parsedPort
		}
	}

	host, resolvedPort, cfg, err := normalizeTCPPingRequest(host, *port, *count, *interval, *timeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	rep, err := tcpping.Run(host, resolvedPort, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *jsonOut {
		printJSON(rep)
		return
	}
	printTCPPing(rep)
}

func runPortScan(args []string) {
	fs := flag.NewFlagSet("portscan", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	portsSpec := fs.String("ports", defaultScanPorts, "comma-separated ports and ranges (for example 22,80,443,8000-8100)")
	timeout := fs.Duration("timeout", defaultProbeTimeout, "per-port timeout")
	concurrency := fs.Int("concurrency", defaultScanWorkers, "maximum parallel connection attempts")
	jsonOut := fs.Bool("json", false, "JSON output")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: netwatcher portscan [flags] <host>")
		fs.PrintDefaults()
	}

	if err := fs.Parse(normalizeArgs(fs, args)); err != nil {
		os.Exit(2)
	}

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(2)
	}

	host, ports, cfg, err := normalizePortScanRequest(fs.Arg(0), *portsSpec, *timeout, *concurrency)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	rep, err := portscan.Run(host, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *jsonOut {
		printPortScanJSON(rep)
		return
	}
	printPortScan(rep, ports)
}

func runDNSLookup(args []string) {
	fs := flag.NewFlagSet("dns", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	timeout := fs.Duration("timeout", dnslookup.DefaultTimeout, "per-query timeout")
	recordType := fs.String("type", "all", "record type: all, a, aaaa, cname, ns, mx, txt, dmarc, soa, srv, caa, dnskey, ds, rrsig, ptr")
	resolver := fs.String("resolver", "auto", "resolver: auto, cloudflare, google, quad9, or host[:port]")
	jsonOut := fs.Bool("json", false, "JSON output")

	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: netwatcher dns [flags] <host-or-ip>")
		fs.PrintDefaults()
	}

	if err := fs.Parse(normalizeArgs(fs, args)); err != nil {
		os.Exit(2)
	}

	if fs.NArg() < 1 {
		fs.Usage()
		os.Exit(2)
	}

	target, err := normalizeHost(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	normalizedTimeout, err := normalizeDNSLookupTimeout(*timeout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	rep, err := dnslookup.Run(target, dnslookup.Config{
		Timeout:    normalizedTimeout,
		RecordType: *recordType,
		Resolver:   *resolver,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *jsonOut {
		printDNSLookupJSON(rep)
		return
	}
	printDNSLookup(rep)
}

func printPing(rep report.Report) {
	fmt.Printf("PING %s (%s)\n", rep.Target, rep.Addr)
	for _, r := range rep.Results {
		if r.Error != "" {
			fmt.Printf("seq=%d %s\n", r.Seq, r.Error)
			continue
		}
		addr := r.Addr
		if addr == "" {
			addr = rep.Addr
		}
		fmt.Printf("reply from %s: seq=%d time=%s\n", addr, r.Seq, formatDuration(r.RTT))
	}
	printSummary(rep)
}

func printTCPPing(rep report.Report) {
	fmt.Printf("TCP PING %s (%s)\n", rep.Target, rep.Addr)
	for _, r := range rep.Results {
		if r.Error != "" {
			fmt.Printf("seq=%d %s\n", r.Seq, r.Error)
			continue
		}
		fmt.Printf("seq=%d time=%s\n", r.Seq, formatDuration(r.RTT))
	}
	printSummary(rep)
}

func printPortScan(rep portscan.Report, ports []int) {
	fmt.Printf("TCP PORT SCAN %s", rep.Target)
	if rep.Addr != "" {
		fmt.Printf(" (%s)", rep.Addr)
	}
	fmt.Println()

	for _, result := range rep.Results {
		if result.State == "open" {
			fmt.Printf("port=%d state=open time=%s", result.Port, formatDuration(result.RTT))
		} else {
			fmt.Printf("port=%d state=%s", result.Port, result.State)
		}
		if result.Addr != "" {
			fmt.Printf(" addr=%s", result.Addr)
		}
		if result.Error != "" && result.State != "open" {
			fmt.Printf(" error=%s", result.Error)
		}
		fmt.Println()
	}

	fmt.Printf("--- %s tcp-portscan statistics ---\n", rep.Target)
	fmt.Printf("%d ports scanned, %d open, %d closed, %d timeout, completed in %s\n",
		rep.Summary.Scanned,
		rep.Summary.Open,
		rep.Summary.Closed,
		rep.Summary.Timeout,
		rep.Summary.Duration.Round(time.Millisecond),
	)
	if len(ports) > 0 {
		fmt.Printf("%d requested ports\n", len(ports))
	}
}

func printSummary(rep report.Report) {
	fmt.Printf("--- %s %s statistics ---\n", rep.Target, rep.Protocol)
	fmt.Printf("%d probes, %d replies, %.1f%% loss\n", rep.Summary.Sent, rep.Summary.Recv, rep.Summary.Loss)
	if rep.Summary.Recv > 0 {
		fmt.Printf("rtt min/avg/max/stddev = %s/%s/%s/%s\n",
			formatDuration(rep.Summary.Min),
			formatDuration(rep.Summary.Avg),
			formatDuration(rep.Summary.Max),
			formatDuration(rep.Summary.StdDev),
		)
	}
}

func printJSON(rep report.Report) {
	jr := toJSONReport(rep)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(jr)
}

func printPortScanJSON(rep portscan.Report) {
	jr := toJSONPortScanReport(rep)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(jr)
}

func printDNSLookup(rep dnslookup.Report) {
	fmt.Printf("DNS LOOKUP %s\n", rep.Target)
	fmt.Printf("kind=%s duration=%.1fms\n", rep.Kind, rep.Summary.DurationMs)

	printDNSSection := func(label string, section dnslookup.StringSection) {
		if len(section.Values) == 0 && section.Error == "" {
			return
		}
		fmt.Printf("%s:\n", label)
		if len(section.Values) > 0 {
			for _, value := range section.Values {
				fmt.Printf("  - %s\n", value)
			}
		}
		if section.Error != "" {
			fmt.Printf("  ! %s\n", section.Error)
		}
	}

	printDNSSection("A", rep.A)
	printDNSSection("AAAA", rep.AAAA)
	printDNSSection("CNAME", rep.CNAME)
	printDNSSection("NS", rep.NS)
	if len(rep.MX.Records) > 0 || rep.MX.Error != "" {
		fmt.Println("MX:")
		for _, record := range rep.MX.Records {
			fmt.Printf("  - %d %s\n", record.Pref, record.Host)
		}
		if rep.MX.Error != "" {
			fmt.Printf("  ! %s\n", rep.MX.Error)
		}
	}
	printDNSSection("TXT", rep.TXT)
	printDNSSection("DMARC", rep.DMARC)
	if rep.SOA.Record != nil || rep.SOA.Error != "" {
		fmt.Println("SOA:")
		if rep.SOA.Record != nil {
			fmt.Printf("  - ns=%s mbox=%s serial=%d refresh=%d retry=%d expire=%d minttl=%d\n",
				rep.SOA.Record.NS,
				rep.SOA.Record.MBox,
				rep.SOA.Record.Serial,
				rep.SOA.Record.Refresh,
				rep.SOA.Record.Retry,
				rep.SOA.Record.Expire,
				rep.SOA.Record.MinTTL,
			)
		}
		if rep.SOA.Error != "" {
			fmt.Printf("  ! %s\n", rep.SOA.Error)
		}
	}
	if len(rep.SRV.Records) > 0 || rep.SRV.Error != "" {
		fmt.Println("SRV:")
		for _, record := range rep.SRV.Records {
			fmt.Printf("  - %d %d %d %s\n", record.Priority, record.Weight, record.Port, record.Target)
		}
		if rep.SRV.Error != "" {
			fmt.Printf("  ! %s\n", rep.SRV.Error)
		}
	}
	if len(rep.CAA.Records) > 0 || rep.CAA.Error != "" {
		fmt.Println("CAA:")
		for _, record := range rep.CAA.Records {
			fmt.Printf("  - flag=%d tag=%s value=%s\n", record.Flag, record.Tag, record.Value)
		}
		if rep.CAA.Error != "" {
			fmt.Printf("  ! %s\n", rep.CAA.Error)
		}
	}
	printDNSSection("DNSKEY", rep.DNSKEY)
	printDNSSection("DS", rep.DS)
	printDNSSection("RRSIG", rep.RRSIG)
	if len(rep.PTR) > 0 {
		fmt.Println("PTR:")
		for _, record := range rep.PTR {
			if len(record.Names) > 0 {
				fmt.Printf("  %s\n", record.Address)
				for _, name := range record.Names {
					fmt.Printf("    - %s\n", name)
				}
				continue
			}
			fmt.Printf("  %s ! %s\n", record.Address, record.Error)
		}
	}
	fmt.Printf("analysis: mail_ready=%t spf=%t dmarc=%t mx=%t caa=%t dnssec=%t\n",
		rep.Analysis.MailReady,
		rep.Analysis.SPFPresent,
		rep.Analysis.DMARCPresent,
		rep.Analysis.MXPresent,
		rep.Analysis.CAAPresent,
		rep.Analysis.DNSSECPresent,
	)
}

func printDNSLookupJSON(rep dnslookup.Report) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(rep)
}

func formatDuration(d time.Duration) string {
	ms := float64(d) / float64(time.Millisecond)
	return fmt.Sprintf("%.3fms", ms)
}
