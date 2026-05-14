const API_BASE = '';

async function request<T>(path: string, opts?: RequestInit): Promise<T> {
  const resp = await fetch(`${API_BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...opts,
  });
  if (!resp.ok) {
    const err = await resp.json().catch(() => ({ error: resp.statusText }));
    throw new Error(err.error || resp.statusText);
  }
  return resp.json();
}

export interface Site {
  id: string;
  name: string;
  base_url: string;
  protocol: string;
  api_key: string;
  models: string[];
  balance_api?: string;
}

export interface Rule {
  model_pattern: string;
  site: string;
  fallback?: string;
}

export interface Profile {
  name: string;
  rules: Rule[];
}

export interface RoutingCurrent {
  active_profile: string;
  profile: Profile | null;
  sites_count: number;
}

export interface RequestLog {
  id: number;
  request_id: string;
  timestamp: string;
  site_id: string;
  model: string;
  protocol: string;
  input_tokens: number;
  output_tokens: number;
  latency_ms: number;
  status_code: number;
  is_stream: boolean;
  error: string;
  client_ip: string;
  cost: number;
}

export interface Stats {
  total_requests: number;
  total_input_tokens: number;
  total_output_tokens: number;
  total_cost: number;
}

export interface AppConfig {
  proxy: { listen: string; web_listen: string };
  security: {
    api_key_encryption: boolean;
    access_token: string;
    allowed_ips: string[];
    rate_limit: { global_rpm: number; per_site_rpm: Record<string, number> };
  };
  logging: { enabled: boolean; max_days: number; log_body: boolean };
}

export const api = {
  getRoutingCurrent: () => request<RoutingCurrent>('/api/v1/routing/current'),
  getSites: () => request<{ sites: Site[] }>('/api/v1/sites').then(r => r.sites),
  getProfiles: () => request<{ profiles: Profile[]; active_profile: string }>('/api/v1/profiles'),
  switchProfile: (profile: string) =>
    request<{ success: boolean }>('/api/v1/routing/switch', {
      method: 'POST',
      body: JSON.stringify({ profile }),
    }),
  switchSite: (site: string) =>
    request<{ success: boolean }>('/api/v1/routing/switch', {
      method: 'POST',
      body: JSON.stringify({ site }),
    }),
  switchModel: (model: string, site_for_model: string) =>
    request<{ success: boolean }>('/api/v1/routing/switch', {
      method: 'POST',
      body: JSON.stringify({ model, site_for_model }),
    }),
  getLogs: (params?: string) =>
    request<{ logs: RequestLog[] }>(`/api/v1/logs${params ? '?' + params : ''}`).then(r => r.logs),
  getStats: (days?: number) =>
    request<Stats>(`/api/v1/stats${days ? '?days=' + days : ''}`),
  addSite: (site: Partial<Site> & { id: string; api_key: string }) =>
    request<{ success: boolean; site_id: string }>('/api/v1/sites/add', {
      method: 'POST',
      body: JSON.stringify(site),
    }),
  updateSite: (site: Site) =>
    request<{ success: boolean }>('/api/v1/sites/update', {
      method: 'POST',
      body: JSON.stringify(site),
    }),
  deleteSite: (id: string) =>
    request<{ success: boolean }>('/api/v1/sites/delete', {
      method: 'POST',
      body: JSON.stringify({ id }),
    }),
  createProfile: (name: string) =>
    request<{ success: boolean }>('/api/v1/profiles/create', {
      method: 'POST',
      body: JSON.stringify({ name }),
    }),
  deleteProfile: (name: string) =>
    request<{ success: boolean }>('/api/v1/profiles/delete', {
      method: 'POST',
      body: JSON.stringify({ name }),
    }),
  addProfileRule: (profile: string, rule: Rule) =>
    request<{ success: boolean }>('/api/v1/profiles/rules/add', {
      method: 'POST',
      body: JSON.stringify({ profile, ...rule }),
    }),
  deleteProfileRule: (profile: string, index: number) =>
    request<{ success: boolean }>('/api/v1/profiles/rules/delete', {
      method: 'POST',
      body: JSON.stringify({ profile, index }),
    }),
  getConfig: () => request<AppConfig>('/api/v1/config'),
  updateConfig: (updates: Partial<AppConfig>) =>
    request<{ success: boolean }>('/api/v1/config', {
      method: 'PATCH',
      body: JSON.stringify(updates),
    }),
  reloadConfig: () =>
    request<{ success: boolean }>('/api/v1/config/reload', { method: 'POST' }),
  health: () => request<{ status: string }>('/api/v1/health'),
};
