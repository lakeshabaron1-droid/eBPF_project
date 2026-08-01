import React, { useEffect, useRef, useState } from 'react';

const REASON_LABELS = {
  1: 'BLOCKLIST',
  2: 'RATE_LIMIT',
};

const PROTOCOL_LABELS = {
  6: 'TCP',
  17: 'UDP',
  1: 'ICMP',
};

export default function EventLog({ dropEvents = [] }) {
  const [events, setEvents] = useState([]);
  const [flashIndex, setFlashIndex] = useState(-1);
  const tableRef = useRef(null);

  useEffect(() => {
    if (!dropEvents || dropEvents.length === 0) return;

    setEvents((prev) => {
      const newEvents = dropEvents.map((ev) => ({
        ...ev,
        time: new Date().toLocaleTimeString(),
        id: Math.random().toString(36).substring(2, 9),
      }));
      const merged = [...newEvents, ...prev];
      return merged.slice(0, 100);
    });

    setFlashIndex(0);
    const timeout = setTimeout(() => setFlashIndex(-1), 600);
    return () => clearTimeout(timeout);
  }, [dropEvents]);

  return (
    <div className="table-container" ref={tableRef}>
      <table className="data-table">
        <thead>
          <tr>
            <th>Time</th>
            <th>Source IP</th>
            <th>Dest IP</th>
            <th>Proto</th>
            <th>Ports</th>
            <th>Reason</th>
          </tr>
        </thead>
        <tbody>
          {events.length === 0 ? (
            <tr>
              <td colSpan={6} style={{ textAlign: 'center', color: 'var(--text-muted)', padding: '24px' }}>
                No drop events recorded
              </td>
            </tr>
          ) : (
            events.map((ev, idx) => (
              <tr
                key={ev.id}
                className={idx === 0 && flashIndex === 0 ? 'row-flash' : ''}
              >
                <td>{ev.time}</td>
                <td>{ev.src_ip}</td>
                <td>{ev.dst_ip}</td>
                <td>{PROTOCOL_LABELS[ev.protocol] || ev.protocol}</td>
                <td>{ev.src_port} : {ev.dst_port}</td>
                <td>
                  <span className="tag tag-drop">
                    {REASON_LABELS[ev.reason] || `CODE_${ev.reason}`}
                  </span>
                </td>
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}
