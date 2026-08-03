const API_BASE = '/api';

async function request(path, options = {}) {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: {
      'Content-Type': 'application/json',
    },
    ...options,
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `Request failed: ${res.status}`);
  }

  return res.json();
}

export async function blockIP(ip) {
  return request('/block', {
    method: 'POST',
    body: JSON.stringify({ ip }),
  });
}

export async function unblockIP(ip) {
  return request(`/block/${ip}`, {
    method: 'DELETE',
  });
}

export async function updateRateLimit(threshold, windowMs) {
  return request('/config/ratelimit', {
    method: 'PUT',
    body: JSON.stringify({ threshold, window_ms: windowMs }),
  });
}

export async function getRoutes() {
  return request('/routes', {
    method: 'GET',
  });
}

export async function getBlockedIPs() {
  return request('/blocked', {
    method: 'GET',
  });
}
