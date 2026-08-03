'use client';

import React, { useState, useEffect, useRef } from 'react';
import { useMetricsStream } from '../hooks/useMetricsStream';

import MetricCard from '../components/MetricCard';
import TrafficChart from '../components/TrafficChart';
import ProtocolChart from '../components/ProtocolChart';
import BlockedIPsChart from '../components/BlockedIPsChart';
import EventLog from '../components/EventLog';
import StatusIndicator from '../components/StatusIndicator';
import ControlPanel from '../components/ControlPanel';

export default function Dashboard() {
  const { data, status } = useMetricsStream('/events');
  const [toast, setToast] = useState(null);
  const [uptime, setUptime] = useState(0);
  const [passedHistory, setPassedHistory] = useState([]);
  const [droppedHistory, setDroppedHistory] = useState([]);
  const toastTimeout = useRef(null);

  useEffect(() => {
    const interval = setInterval(() => {
      setUptime((prev) => prev + 1);
    }, 1000);
    return () => clearInterval(interval);
  }, []);

  useEffect(() => {
    if (!data) return;
    setPassedHistory((prev) => {
      const next = [...prev, data.passed_per_sec || 0];
      return next.length > 20 ? next.slice(-20) : next;
    });
    setDroppedHistory((prev) => {
      const next = [...prev, data.dropped_per_sec || 0];
      return next.length > 20 ? next.slice(-20) : next;
    });
  }, [data]);

  const showToast = (message) => {
    setToast(message);
    if (toastTimeout.current) clearTimeout(toastTimeout.current);
    toastTimeout.current = setTimeout(() => setToast(null), 3000);
  };

  const formatUptime = (seconds) => {
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = seconds % 60;
    return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
  };

  const dropRate = data?.drop_rate != null
    ? (data.drop_rate * 100).toFixed(2) + '%'
    : '0.00%';

  return (
    <>
      <header className="header">
        <div className="header-title">
          <div>
            <h1>eBPF Zero-Trust Gateway</h1>
            <div className="header-subtitle">
              Uptime {formatUptime(uptime)}
            </div>
          </div>
        </div>
        <div className="header-status">
          <StatusIndicator status={status} />
        </div>
      </header>

      <div className="metrics-grid">
        <MetricCard
          title="Packets Passed"
          value={data?.total_passed || 0}
          subtext={`${(data?.passed_per_sec || 0).toFixed(1)} pkt/s`}
          sparklineData={passedHistory}
          color="green"
        />
        <MetricCard
          title="Packets Dropped"
          value={data?.total_dropped || 0}
          subtext={`${(data?.dropped_per_sec || 0).toFixed(1)} pkt/s`}
          sparklineData={droppedHistory}
          color="red"
        />
        <MetricCard
          title="Drop Rate"
          value={dropRate}
          subtext="current window"
          color="amber"
        />
        <MetricCard
          title="Blocked IPs"
          value={data?.top_blocked_ips?.length || 0}
          subtext="unique sources"
          color="purple"
        />
      </div>

      <div className="charts-grid">
        <div className="glass-card">
          <div className="section-title">Traffic Over Time</div>
          <TrafficChart snapshot={data} />
        </div>
        <div className="glass-card">
          <div className="section-title">Protocol Distribution</div>
          <ProtocolChart protocolStats={data?.protocol_stats} />
        </div>
      </div>

      <div className="dashboard-bottom-grid">
        <div className="glass-card">
          <div className="section-title">Top Blocked IPs</div>
          <BlockedIPsChart topBlockedIPs={data?.top_blocked_ips} />
        </div>
        <div className="glass-card">
          <div className="section-title">Drop Events</div>
          <EventLog dropEvents={data?.drop_events} />
        </div>
      </div>

      <div style={{ marginTop: '20px' }}>
        <div className="glass-card">
          <div className="section-title">Gateway Controls</div>
          <ControlPanel onToast={showToast} />
        </div>
      </div>

      {toast && <div className="toast">{toast}</div>}
    </>
  );
}
