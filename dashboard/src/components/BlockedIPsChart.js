import React from 'react';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend,
} from 'chart.js';
import { Bar } from 'react-chartjs-2';

ChartJS.register(
  CategoryScale,
  LinearScale,
  BarElement,
  Title,
  Tooltip,
  Legend
);

export default function BlockedIPsChart({ topBlockedIPs = [] }) {
  const sorted = [...topBlockedIPs].sort((a, b) => a.count - b.count);

  const data = {
    labels: sorted.map((entry) => entry.ip),
    datasets: [
      {
        label: 'Dropped Packets',
        data: sorted.map((entry) => entry.count),
        backgroundColor: sorted.map((_, idx) => {
          const intensity = 0.4 + (idx / Math.max(sorted.length - 1, 1)) * 0.6;
          return `rgba(239, 68, 68, ${intensity})`;
        }),
        borderColor: '#ef4444',
        borderWidth: 1,
        borderRadius: 4,
      },
    ],
  };

  const options = {
    responsive: true,
    maintainAspectRatio: false,
    indexAxis: 'y',
    plugins: {
      legend: {
        display: false,
      },
      tooltip: {
        callbacks: {
          label: function (context) {
            return `${context.parsed.x.toLocaleString()} packets dropped`;
          },
        },
      },
    },
    scales: {
      x: {
        grid: {
          color: 'rgba(255, 255, 255, 0.05)',
        },
        ticks: {
          color: '#6b7280',
          font: {
            family: 'var(--font-mono)',
            size: 10,
          },
        },
      },
      y: {
        grid: {
          display: false,
        },
        ticks: {
          color: '#9ca3af',
          font: {
            family: 'var(--font-mono)',
            size: 11,
          },
        },
      },
    },
  };

  if (sorted.length === 0) {
    return (
      <div className="chart-container" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <span style={{ color: 'var(--text-muted)', fontFamily: 'var(--font-mono)', fontSize: '13px' }}>
          No blocked IPs
        </span>
      </div>
    );
  }

  return (
    <div className="chart-container">
      <Bar data={data} options={options} />
    </div>
  );
}
