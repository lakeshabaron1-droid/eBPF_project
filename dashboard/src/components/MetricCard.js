import React, { useState, useEffect } from 'react';

export default function MetricCard({ title, value, subtext, trend, sparklineData = [], color = 'blue' }) {
  const [displayValue, setDisplayValue] = useState(0);

  useEffect(() => {
    const numericTarget = typeof value === 'number' ? value : parseFloat(value);
    if (isNaN(numericTarget)) return;

    let start = displayValue;
    const end = numericTarget;
    const duration = 500;
    const startTime = performance.now();

    const animate = (currentTime) => {
      const elapsed = currentTime - startTime;
      const progress = Math.min(elapsed / duration, 1);
      const current = Math.floor(start + (end - start) * progress);
      setDisplayValue(current);

      if (progress < 1) {
        requestAnimationFrame(animate);
      }
    };

    requestAnimationFrame(animate);
  }, [value]);

  const renderSparkline = () => {
    if (!sparklineData || sparklineData.length < 2) return null;

    const max = Math.max(...sparklineData, 1);
    const min = Math.min(...sparklineData, 0);
    const range = max - min || 1;
    const width = 120;
    const height = 30;

    const points = sparklineData
      .map((val, idx) => {
        const x = (idx / (sparklineData.length - 1)) * width;
        const y = height - ((val - min) / range) * height;
        return `${x},${y}`;
      })
      .join(' ');

    return (
      <svg width={width} height={height} style={{ overflow: 'visible' }}>
        <polyline
          fill="none"
          stroke={`var(--accent-${color}, #3b82f6)`}
          strokeWidth="2"
          points={points}
        />
      </svg>
    );
  };

  return (
    <div className="glass-card">
      <div className="metric-card-header">
        <span className="metric-title">{title}</span>
        {renderSparkline()}
      </div>
      <div className="metric-value">
        {typeof value === 'number' ? displayValue.toLocaleString() : value}
      </div>
      {subtext && <div className="metric-sub">{subtext}</div>}
    </div>
  );
}
