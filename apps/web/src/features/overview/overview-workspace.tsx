"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { useQueries, useQuery } from "@tanstack/react-query";
import { apiRequest } from "@/lib/api";
import { AlertRule, HistoryPoint, RecentHistoryTarget, RuntimeHealthResponse, SessionResponse } from "@/lib/types";
import { LiveTelemetrySample, subscribeLiveTelemetry } from "@/lib/live-telemetry";
import {
  buildOverviewKpis,
  buildOverviewLatencySeries,
  buildOverviewTargetSnapshots,
  uniqueAlertTargets,
} from "@/features/overview/overview-helpers";
import { buildOverviewDemoData } from "@/features/overview/overview-demo-data";
import { GettingStarted, LatencyCard, OverviewHeader, QuickProbe, StatRow, StatusCard, TargetsTable } from "@/features/overview/overview-view";

type ActiveTool = "icmp" | "tcp" | "portscan" | "dns";


function buildToolHref(tool: ActiveTool, target: string) {
  const value = target.trim();
  const params = new URLSearchParams();
  if (value) {
    params.set("target", value);
  }

  const query = params.toString();
  switch (tool) {
    case "tcp":
      return query ? `/tcp-ping?${query}` : "/tcp-ping";
    case "portscan":
      return query ? `/port-scan?${query}` : "/port-scan";
    case "dns":
      return query ? `/dns-lookup?${query}` : "/dns-lookup";
    default:
      return query ? `/icmp-ping?${query}` : "/icmp-ping";
  }
}

function mergeTrackedRules(alertRules: AlertRule[], recentTargets: RecentHistoryTarget[]) {
  const merged = new Map<string, AlertRule>();

  alertRules.forEach((rule) => {
    merged.set(`${rule.protocol}:${rule.target}:${rule.port ?? 0}`, rule);
  });

  recentTargets.forEach((target, index) => {
    const key = `${target.protocol}:${target.target}:${target.port ?? 0}`;
    if (merged.has(key)) {
      return;
    }

    merged.set(key, {
      id: -(index + 1),
      name: "",
      protocol: target.protocol,
      target: target.target,
      port: target.port ?? null,
      recipient_email: "",
      consecutive_breaches: 1,
      cooldown_minutes: 30,
      notify_recovery: false,
      enabled: true,
      last_state: "healthy",
      current_breach_streak: 0,
      created_at: new Date(target.last_sample_at).toISOString(),
    });
  });

  return Array.from(merged.values()).slice(0, 12);
}

function mergeTrackedRulesFromLiveSamples(rules: AlertRule[], liveSamples: LiveTelemetrySample[]) {
  const merged = new Map<string, AlertRule>();

  rules.forEach((rule) => {
    merged.set(`${rule.protocol}:${rule.target}:${rule.port ?? 0}`, rule);
  });

  liveSamples.forEach((sample, index) => {
    const key = `${sample.protocol}:${sample.target}:${sample.port ?? 0}`;
    if (merged.has(key)) {
      return;
    }

    merged.set(key, {
      id: -(1000 + index + 1),
      name: "",
      protocol: sample.protocol,
      target: sample.target,
      port: sample.port ?? null,
      recipient_email: "",
      consecutive_breaches: 1,
      cooldown_minutes: 30,
      notify_recovery: false,
      enabled: true,
      last_state: sample.error ? "alert" : "healthy",
      current_breach_streak: 0,
      created_at: new Date(sample.ts).toISOString(),
    });
  });

  return Array.from(merged.values()).slice(0, 12);
}

export function OverviewWorkspace() {
  const router = useRouter();
  const [activeTool, setActiveTool] = useState<ActiveTool>("icmp");
  const [target, setTarget] = useState("");
  const [liveSamples, setLiveSamples] = useState<LiveTelemetrySample[]>([]);
  useEffect(() => {
    return subscribeLiveTelemetry((sample) => {
      setLiveSamples((current) => [...current, sample].slice(-96));
    });
  }, []);

  const sessionQuery = useQuery({
    queryKey: ["session"],
    queryFn: () => apiRequest<SessionResponse>("/api/auth/session"),
    staleTime: 60_000,
  });
  const isDemo = sessionQuery.data?.read_only === true;
  const demoData = useMemo(() => buildOverviewDemoData(), []);

  const runtimeHealthQuery = useQuery({
    queryKey: ["runtime-health"],
    queryFn: () => apiRequest<RuntimeHealthResponse>("/api/healthz"),
    refetchInterval: 30_000,
  });

  const alertRulesQuery = useQuery({
    queryKey: ["alert-rules"],
    queryFn: () => apiRequest<AlertRule[]>("/api/alerts/rules"),
    refetchInterval: 20_000,
    enabled: sessionQuery.data?.authenticated === true && !isDemo,
  });

  const recentTargetsQuery = useQuery({
    queryKey: ["recent-history-targets"],
    queryFn: () => apiRequest<RecentHistoryTarget[]>("/api/history/recent-targets?limit=12"),
    refetchInterval: 20_000,
    enabled: sessionQuery.data?.authenticated === true && !isDemo,
  });

  const alertRulesSource = useMemo(
    () => (isDemo ? demoData.alertRules : alertRulesQuery.data ?? []),
    [alertRulesQuery.data, demoData.alertRules, isDemo],
  );
  const recentTargetsSource = useMemo(
    () => (isDemo ? demoData.recentTargets : recentTargetsQuery.data ?? []),
    [demoData.recentTargets, isDemo, recentTargetsQuery.data],
  );

  const trackedRules = useMemo(
    () => mergeTrackedRules(uniqueAlertTargets(alertRulesSource), recentTargetsSource),
    [alertRulesSource, recentTargetsSource],
  );
  const trackedRulesWithLive = useMemo(() => mergeTrackedRulesFromLiveSamples(trackedRules, liveSamples), [trackedRules, liveSamples]);

  const historyQueries = useQueries({
    queries: isDemo ? [] : trackedRulesWithLive.map((rule) => {
      const params = new URLSearchParams({
        type: rule.protocol,
        host: rule.target,
        limit: "24",
      });
      if (rule.protocol === "tcpping" && rule.port) {
        params.set("port", String(rule.port));
      }
      return {
        queryKey: ["overview-history", rule.protocol, rule.target, rule.port ?? 0],
        queryFn: () => apiRequest<HistoryPoint[]>(`/api/history?${params.toString()}`),
        refetchInterval: 20_000,
      };
    }),
  });

  const historyMap = useMemo(() => {
    if (isDemo) {
      return demoData.historyMap;
    }
    const map = new Map<string, HistoryPoint[]>();
    trackedRulesWithLive.forEach((rule, index) => {
      const key = `${rule.protocol}:${rule.target}:${rule.port ?? 0}`;
      map.set(key, historyQueries[index]?.data ?? []);
    });
    return map;
  }, [demoData.historyMap, historyQueries, isDemo, trackedRulesWithLive]);

  const liveSampleMap = useMemo(() => {
    const map = new Map<string, LiveTelemetrySample>();
    liveSamples.forEach((sample) => {
      map.set(`${sample.protocol}:${sample.target}:${sample.port ?? 0}`, sample);
    });
    return map;
  }, [liveSamples]);

  const targetSnapshots = useMemo(
    () =>
      buildOverviewTargetSnapshots(trackedRulesWithLive, historyMap, recentTargetsSource).map((snapshot) => {
        const liveSample = liveSampleMap.get(snapshot.id);
        if (!liveSample) {
          return snapshot;
        }
        return {
          ...snapshot,
          healthState:
            liveSample.error && snapshot.healthState !== "critical"
              ? "warning"
              : snapshot.healthState === "idle" && !liveSample.error
                ? "stable"
                : snapshot.healthState,
          lastSampleAt: Math.max(snapshot.lastSampleAt ?? 0, liveSample.ts),
          lastRttMs: liveSample.rttMs ?? snapshot.lastRttMs,
        };
      }),
    [trackedRulesWithLive, historyMap, recentTargetsSource, liveSampleMap],
  );
  const latencySeries = useMemo(() => {
    const livePoints = liveSamples
      .filter((sample) => typeof sample.rttMs === "number")
      .map((sample) => ({
        key: `${sample.protocol}:${sample.target}:${sample.port ?? 0}`,
        ts: sample.ts,
        rttMs: sample.rttMs,
      }));

    return [...buildOverviewLatencySeries(historyMap), ...livePoints]
      .sort((a, b) => a.ts - b.ts)
      .slice(-32);
  }, [historyMap, liveSamples]);
  const kpis = useMemo(
    () => buildOverviewKpis(trackedRulesWithLive, runtimeHealthQuery.data, targetSnapshots),
    [trackedRulesWithLive, runtimeHealthQuery.data, targetSnapshots],
  );
  function runActiveTool() {
    router.push(buildToolHref(activeTool, target));
  }

  const alertRuleCount = alertRulesSource.length;
  const hasData = targetSnapshots.length > 0 || latencySeries.length > 0;
  const degraded = kpis.critical + kpis.warning;

  return (
    <div className="grid grid-cols-1 gap-6 [&>*]:min-w-0">
      <OverviewHeader health={runtimeHealthQuery.data} />
      <QuickProbe
        tool={activeTool}
        onToolChange={setActiveTool}
        target={target}
        onTargetChange={setTarget}
        onRun={runActiveTool}
        disabled={isDemo}
      />
      {hasData ? (
        <>
          <StatRow tracked={targetSnapshots.length} healthy={kpis.stable} degraded={degraded} live={kpis.liveStreams} />
          <div className="grid gap-6 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
            <LatencyCard points={latencySeries} />
            <StatusCard targets={targetSnapshots} alertRuleCount={alertRuleCount} />
          </div>
          {targetSnapshots.length > 0 ? <TargetsTable targets={targetSnapshots} /> : null}
        </>
      ) : (
        <GettingStarted hasAlertRules={alertRuleCount > 0} />
      )}
    </div>
  );
}
