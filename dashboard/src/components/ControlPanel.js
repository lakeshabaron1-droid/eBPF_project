import React, { useState, useEffect } from 'react';
import { blockIP, unblockIP, updateRateLimit, getRoutes, getBlockedIPs } from '../hooks/useGatewayAPI';

export default function ControlPanel({ onToast }) {
  const [ipInput, setIpInput] = useState('');
  const [threshold, setThreshold] = useState(100);
  const [windowMs, setWindowMs] = useState(1000);
  const [routes, setRoutes] = useState([]);
  const [blockedIPs, setBlockedIPs] = useState([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    getRoutes()
      .then(setRoutes)
      .catch(() => {});

    getBlockedIPs()
      .then((data) => setBlockedIPs(data.blocked_ips || []))
      .catch(() => {});
  }, []);

  const handleBlock = async () => {
    if (!ipInput.trim()) return;
    setLoading(true);
    try {
      await blockIP(ipInput.trim());
      onToast(`Blocked ${ipInput.trim()}`);
      setBlockedIPs((prev) => [...prev, ipInput.trim()]);
      setIpInput('');
    } catch (err) {
      onToast(`Failed: ${err.message}`);
    }
    setLoading(false);
  };

  const handleUnblock = async (ip) => {
    try {
      await unblockIP(ip);
      onToast(`Unblocked ${ip}`);
      setBlockedIPs((prev) => prev.filter((i) => i !== ip));
    } catch (err) {
      onToast(`Failed: ${err.message}`);
    }
  };

  const handleRateLimit = async () => {
    setLoading(true);
    try {
      await updateRateLimit(threshold, windowMs);
      onToast(`Rate limit updated: ${threshold} req/${windowMs}ms`);
    } catch (err) {
      onToast(`Failed: ${err.message}`);
    }
    setLoading(false);
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
      <div className="control-form">
        <div className="section-title">IP Blocklist</div>
        <div style={{ display: 'flex', gap: '8px' }}>
          <input
            className="input-field"
            type="text"
            placeholder="192.168.1.100"
            value={ipInput}
            onChange={(e) => setIpInput(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleBlock()}
            style={{ flex: 1 }}
          />
          <button className="btn btn-danger" onClick={handleBlock} disabled={loading}>
            Block
          </button>
        </div>
        {blockedIPs.length > 0 && (
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px', marginTop: '8px' }}>
            {blockedIPs.map((ip) => (
              <div
                key={ip}
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: '6px',
                  padding: '4px 10px',
                  background: 'rgba(239, 68, 68, 0.1)',
                  border: '1px solid rgba(239, 68, 68, 0.3)',
                  borderRadius: 'var(--radius-sm)',
                  fontSize: '12px',
                  fontFamily: 'var(--font-mono)',
                  color: 'var(--accent-red)',
                }}
              >
                {ip}
                <span
                  onClick={() => handleUnblock(ip)}
                  style={{ cursor: 'pointer', fontWeight: 'bold', opacity: 0.7 }}
                >
                  x
                </span>
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="control-form">
        <div className="section-title">Rate Limiting</div>
        <div className="form-group">
          <label>Threshold (requests per window)</label>
          <div className="slider-container">
            <input
              className="slider"
              type="range"
              min="10"
              max="10000"
              step="10"
              value={threshold}
              onChange={(e) => setThreshold(parseInt(e.target.value))}
            />
            <span style={{ fontFamily: 'var(--font-mono)', fontSize: '13px', minWidth: '60px', textAlign: 'right' }}>
              {threshold}
            </span>
          </div>
        </div>
        <div className="form-group">
          <label>Window (ms)</label>
          <div className="slider-container">
            <input
              className="slider"
              type="range"
              min="100"
              max="60000"
              step="100"
              value={windowMs}
              onChange={(e) => setWindowMs(parseInt(e.target.value))}
            />
            <span style={{ fontFamily: 'var(--font-mono)', fontSize: '13px', minWidth: '60px', textAlign: 'right' }}>
              {windowMs}ms
            </span>
          </div>
        </div>
        <button className="btn btn-primary" onClick={handleRateLimit} disabled={loading}>
          Apply Rate Limit
        </button>
      </div>

      {routes.length > 0 && (
        <div>
          <div className="section-title">Configured Routes</div>
          <div className="table-container">
            <table className="data-table">
              <thead>
                <tr>
                  <th>Path</th>
                  <th>Upstream</th>
                  <th>Auth</th>
                  <th>Scopes</th>
                </tr>
              </thead>
              <tbody>
                {routes.map((route, idx) => (
                  <tr key={idx}>
                    <td>{route.Path || route.path}</td>
                    <td>{route.Upstream || route.upstream}</td>
                    <td>
                      <span className={`tag ${(route.AuthRequired || route.auth_required) ? 'tag-drop' : 'tag-pass'}`}>
                        {(route.AuthRequired || route.auth_required) ? 'Required' : 'Open'}
                      </span>
                    </td>
                    <td>{(route.RequiredScopes || route.required_scopes || []).join(', ') || '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
