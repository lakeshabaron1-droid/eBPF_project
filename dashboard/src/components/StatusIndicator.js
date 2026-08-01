import React from 'react';

const STATUS_CONFIG = {
  connected: {
    label: 'Connected',
    className: '',
  },
  connecting: {
    label: 'Reconnecting',
    className: 'reconnecting',
  },
  error: {
    label: 'Error',
    className: 'error',
  },
};

export default function StatusIndicator({ status = 'connecting' }) {
  const config = STATUS_CONFIG[status] || STATUS_CONFIG.connecting;

  return (
    <div className={`status-badge ${config.className}`}>
      <span className="pulse-dot"></span>
      {config.label}
    </div>
  );
}
