import { useState, useEffect } from 'react';

export function useMetricsStream(url = '/events') {
  const [data, setData] = useState(null);
  const [status, setStatus] = useState('connecting');

  useEffect(() => {
    let eventSource = null;
    let reconnectTimeout = null;

    const connect = () => {
      setStatus('connecting');
      eventSource = new EventSource(url);

      eventSource.onopen = () => {
        setStatus('connected');
      };

      eventSource.onmessage = (event) => {
        try {
          const parsed = JSON.parse(event.data);
          setData(parsed);
        } catch (e) {
        }
      };

      eventSource.onerror = () => {
        setStatus('error');
        eventSource.close();
        reconnectTimeout = setTimeout(connect, 3000);
      };
    };

    connect();

    return () => {
      if (eventSource) {
        eventSource.close();
      }
      if (reconnectTimeout) {
        clearTimeout(reconnectTimeout);
      }
    };
  }, [url]);

  return { data, status };
}
