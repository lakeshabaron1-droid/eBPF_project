import React from 'react';
import { Chart as ChartJS, ArcElement, Tooltip, Legend } from 'chart.js';
import { Doughnut } from 'react-chartjs-2';

ChartJS.register(ArcElement, Tooltip, Legend);

export default function ProtocolChart({ protocolStats = {} }) {
  const labels = Object.keys(protocolStats).length > 0
    ? Object.keys(protocolStats)
    : ['TCP', 'UDP', 'ICMP', 'Other'];

  const dataValues = Object.keys(protocolStats).length > 0
    ? Object.values(protocolStats)
    : [0, 0, 0, 0];

  const data = {
    labels,
    datasets: [
      {
        data: dataValues,
        backgroundColor: [
          '#3b82f6',
          '#8b5cf6',
          '#06b6d4',
          '#6b7280',
        ],
        borderColor: '#111827',
        borderWidth: 2,
      },
    ],
  };

  const options = {
    responsive: true,
    maintainAspectRatio: false,
    cutout: '70%',
    plugins: {
      legend: {
        position: 'bottom',
        labels: {
          color: '#9ca3af',
          font: {
            family: 'var(--font-mono)',
            size: 11,
          },
          padding: 16,
        },
      },
    },
  };

  return (
    <div className="chart-container">
      <Doughnut data={data} options={options} />
    </div>
  );
}
