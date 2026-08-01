import React, { useEffect, useState } from 'react';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js';
import { Line } from 'react-chartjs-2';

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
);

export default function TrafficChart({ snapshot }) {
  const [history, setHistory] = useState([]);

  useEffect(() => {
    if (!snapshot) return;

    setHistory((prev) => {
      const next = [
        ...prev,
        {
          time: new Date(snapshot.timestamp || Date.now()).toLocaleTimeString(),
          passed: snapshot.passed_per_sec || 0,
          dropped: snapshot.dropped_per_sec || 0,
        },
      ];
      if (next.length > 60) return next.slice(next.length - 60);
      return next;
    });
  }, [snapshot]);

  const data = {
    labels: history.map((item) => item.time),
    datasets: [
      {
        label: 'Passed (pkts/s)',
        data: history.map((item) => item.passed),
        borderColor: '#10b981',
        backgroundColor: 'rgba(16, 185, 129, 0.1)',
        fill: true,
        tension: 0.4,
        pointRadius: 0,
      },
      {
        label: 'Dropped (pkts/s)',
        data: history.map((item) => item.dropped),
        borderColor: '#ef4444',
        backgroundColor: 'rgba(239, 68, 68, 0.1)',
        fill: true,
        tension: 0.4,
        pointRadius: 0,
      },
    ],
  };

  const options = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        position: 'top',
        labels: {
          color: '#9ca3af',
          font: {
            family: 'var(--font-mono)',
          },
        },
      },
      tooltip: {
        mode: 'index',
        intersect: false,
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
          maxTicksLimit: 10,
        },
      },
      y: {
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
    },
  };

  return (
    <div className="chart-container">
      <Line data={data} options={options} />
    </div>
  );
}
