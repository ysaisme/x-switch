import { useState, useEffect, useRef, useCallback } from 'react';
import { api } from './api/client';
import type { RoutingCurrent, Site, Profile, Rule, Stats, RequestLog, AppConfig } from './api/client';

type Page = 'dashboard' | 'switch' | 'sites' | 'logs' | 'stats' | 'settings';
type Theme = 'light' | 'dark' | 'system';

function getEffectiveTheme(theme: Theme): 'light' | 'dark' {
  if (theme === 'system') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }
  return theme;
}

export default function App() {
  const [page, setPage] = useState<Page>('dashboard');
  const [routing, setRouting] = useState<RoutingCurrent | null>(null);
  const [sites, setSites] = useState<Site[]>([]);
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [theme, setTheme] = useState<Theme>(() => {
    return (localStorage.getItem('mswitch-theme') as Theme) || 'system';
  });

  const applyTheme = useCallback((t: Theme) => {
    const effective = getEffectiveTheme(t);
    document.documentElement.classList.toggle('dark', effective === 'dark');
  }, []);

  useEffect(() => {
    applyTheme(theme);
    localStorage.setItem('mswitch-theme', theme);

    if (theme === 'system') {
      const mq = window.matchMedia('(prefers-color-scheme: dark)');
      const handler = () => applyTheme('system');
      mq.addEventListener('change', handler);
      return () => mq.removeEventListener('change', handler);
    }
  }, [theme, applyTheme]);

  const refresh = async () => {
    try {
      setLoading(true);
      setError('');
      const [r, s, p] = await Promise.all([
        api.getRoutingCurrent(),
        api.getSites(),
        api.getProfiles(),
      ]);
      setRouting(r);
      setSites(s);
      setProfiles(p.profiles);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { refresh(); }, []);

  const navItems: { key: Page; label: string }[] = [
    { key: 'dashboard', label: '仪表盘' },
    { key: 'switch', label: '切换' },
    { key: 'sites', label: '站点' },
    { key: 'logs', label: '日志' },
    { key: 'stats', label: '统计' },
    { key: 'settings', label: '设置' },
  ];

  return (
    <div className="flex h-screen bg-[var(--bg-base)] text-[var(--text-primary)]">
      <nav className="w-56 bg-[var(--bg-sidebar)] backdrop-blur-sm flex flex-col shrink-0 border-r border-[var(--border-subtle)]">
        <div className="p-5 pb-4">
          <h1 className="text-xl font-semibold text-[var(--text-primary)] tracking-tight">mswitch</h1>
          <p className="text-xs text-[var(--text-muted)] mt-1.5">
            {routing ? routing.active_profile : 'loading...'}
          </p>
        </div>
        <div className="flex-1 px-3 py-1 space-y-0.5">
          {navItems.map(item => (
            <button
              key={item.key}
              onClick={() => setPage(item.key)}
              className={`w-full text-left px-3.5 py-2.5 text-sm rounded-xl transition-all ${
                page === item.key
                  ? 'bg-[var(--bg-hover)] text-[var(--text-primary)] font-medium'
                  : 'text-[var(--text-muted)] hover:text-[var(--text-secondary)] hover:bg-[var(--bg-card)]'
              }`}
            >
              {item.label}
            </button>
          ))}
        </div>
        <div className="p-3">
          <button
            onClick={refresh}
            className="w-full px-3 py-2 text-xs text-[var(--text-muted)] hover:text-[var(--text-secondary)] hover:bg-[var(--bg-card)] rounded-xl transition-all"
          >
            刷新
          </button>
        </div>
      </nav>

      <main className="flex-1 overflow-auto">
        {error && (
          <div className="m-6 p-4 bg-[var(--accent-red-bg)] rounded-2xl text-[var(--accent-red-text)] text-sm">
            {error}
          </div>
        )}

        {loading && !routing ? (
          <div className="flex items-center justify-center h-full text-[var(--text-muted)]">加载中...</div>
        ) : (
          <>
            {page === 'dashboard' && <DashboardPage routing={routing} sites={sites} onAddSite={() => setPage('sites')} />}
            {page === 'switch' && <SwitchPage routing={routing} sites={sites} profiles={profiles} onSwitch={refresh} />}
            {page === 'sites' && <SitesPage sites={sites} onRefresh={refresh} />}
            {page === 'logs' && <LogsPage />}
            {page === 'stats' && <StatsPage />}
            {page === 'settings' && <SettingsPage theme={theme} onThemeChange={setTheme} />}
          </>
        )}
      </main>
    </div>
  );
}

function DashboardPage({ routing, sites, onAddSite }: { routing: RoutingCurrent | null; sites: Site[]; onAddSite: () => void }) {
  return (
    <div className="p-8 space-y-8">
      <h2 className="text-2xl font-semibold tracking-tight">仪表盘</h2>

      <div className="grid grid-cols-3 gap-5">
        <Card title="站点数" value={String(sites.length)} />
        <Card title="活跃Profile" value={routing?.active_profile || '-'} />
        <Card title="路由规则" value={String(routing?.profile?.rules?.length || 0)} />
      </div>

      <div>
        <h3 className="text-base font-medium mb-4 text-[var(--text-secondary)]">站点列表</h3>
        {sites.length > 0 ? (
        <div className="grid grid-cols-2 gap-4">
          {sites.map(site => (
            <div key={site.id} className="bg-[var(--bg-card)] rounded-2xl p-5 hover:bg-[var(--bg-hover)] transition-colors">
              <div className="flex items-center justify-between mb-3">
                <span className="font-medium">{site.name}</span>
                <span className="text-xs px-2.5 py-1 bg-[var(--bg-input)] rounded-lg">{site.protocol}</span>
              </div>
              <p className="text-xs text-[var(--text-muted)] mb-3">{site.base_url}</p>
              <div className="flex flex-wrap gap-1.5">
                {site.models.map((m: string) => (
                  <span key={m} className="text-xs px-2 py-0.5 bg-[var(--accent-blue-bg)] text-[var(--accent-blue-text)] rounded-lg">
                    {m}
                  </span>
                ))}
              </div>
            </div>
          ))}
        </div>
        ) : (
          <div className="bg-[var(--bg-card)] border border-dashed border-[var(--dashed-border)] rounded-2xl p-10 text-center">
            <p className="text-[var(--text-muted)] mb-4">还没有添加任何 API 站点</p>
            <button
              onClick={onAddSite}
              className="px-5 py-2.5 bg-[var(--accent-blue)] hover:opacity-90 rounded-xl text-sm text-white transition-colors"
            >
              添加第一个站点
            </button>
          </div>
        )}
      </div>

      <div>
        <h3 className="text-base font-medium mb-4 text-[var(--text-secondary)]">当前路由</h3>
        <div className="bg-[var(--bg-card)] rounded-2xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--border-subtle)]">
                <th className="text-left px-5 py-3 text-[var(--text-muted)] font-medium">模型匹配</th>
                <th className="text-left px-5 py-3 text-[var(--text-muted)] font-medium">目标站点</th>
                <th className="text-left px-5 py-3 text-[var(--text-muted)] font-medium">Fallback</th>
              </tr>
            </thead>
            <tbody>
              {routing?.profile?.rules?.map((rule: Rule, i: number) => (
                <tr key={i} className="border-b border-[var(--border-subtle)]">
                  <td className="px-5 py-3 font-mono text-[var(--accent-blue-text)]">{rule.model_pattern}</td>
                  <td className="px-5 py-3">{rule.site}</td>
                  <td className="px-5 py-3 text-[var(--text-muted)]">{rule.fallback || '-'}</td>
                </tr>
              ))}
              {(!routing?.profile?.rules?.length) && (
                <tr><td colSpan={3} className="px-5 py-6 text-center text-[var(--text-muted)]">无路由规则</td></tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}

function SwitchPage({ routing, sites, profiles, onSwitch }: {
  routing: RoutingCurrent | null; sites: Site[]; profiles: Profile[]; onSwitch: () => void;
}) {
  const [switching, setSwitching] = useState(false);
  const [msg, setMsg] = useState('');
  const [newProfileName, setNewProfileName] = useState('');
  const [newRule, setNewRule] = useState({ model_pattern: '', site: '', fallback: '' });
  const [selectedProfile, setSelectedProfile] = useState('');

  const handleSwitch = async (type: 'profile' | 'site', value: string) => {
    try {
      setSwitching(true);
      setMsg('');
      if (type === 'profile') await api.switchProfile(value);
      else await api.switchSite(value);
      setMsg(`已切换到 ${value}`);
      onSwitch();
    } catch (e: any) {
      setMsg(`切换失败: ${e.message}`);
    } finally {
      setSwitching(false);
    }
  };

  const handleCreateProfile = async () => {
    if (!newProfileName.trim()) return;
    try {
      await api.createProfile(newProfileName.trim());
      setNewProfileName('');
      setMsg(`Profile "${newProfileName.trim()}" 已创建`);
      onSwitch();
    } catch (e: any) {
      setMsg(`创建失败: ${e.message}`);
    }
  };

  const handleDeleteProfile = async (name: string) => {
    if (!confirm(`确定删除 Profile "${name}"？`)) return;
    try {
      await api.deleteProfile(name);
      setMsg(`Profile "${name}" 已删除`);
      onSwitch();
    } catch (e: any) {
      setMsg(`删除失败: ${e.message}`);
    }
  };

  const handleAddRule = async () => {
    if (!selectedProfile || !newRule.model_pattern || !newRule.site) return;
    try {
      await api.addProfileRule(selectedProfile, newRule);
      setNewRule({ model_pattern: '', site: '', fallback: '' });
      setMsg('规则已添加');
      onSwitch();
    } catch (e: any) {
      setMsg(`添加失败: ${e.message}`);
    }
  };

  const handleDeleteRule = async (profile: string, index: number) => {
    try {
      await api.deleteProfileRule(profile, index);
      setMsg('规则已删除');
      onSwitch();
    } catch (e: any) {
      setMsg(`删除失败: ${e.message}`);
    }
  };

  return (
    <div className="p-8 space-y-8">
      <h2 className="text-2xl font-semibold tracking-tight">切换中心</h2>

      {msg && (
        <div className="p-4 bg-[var(--accent-blue-bg)] rounded-2xl text-[var(--accent-blue-text)] text-sm">{msg}</div>
      )}

      <div className="space-y-5">
        <div className="bg-[var(--bg-card)] rounded-2xl p-5">
          <h3 className="font-medium mb-2">快速切换站点</h3>
          <p className="text-sm text-[var(--text-muted)] mb-4">将所有请求路由到指定站点</p>
          <div className="flex flex-wrap gap-2.5">
            {sites.map(site => (
              <button
                key={site.id}
                onClick={() => handleSwitch('site', site.id)}
                disabled={switching}
                className="px-4 py-2.5 bg-[var(--accent-blue)] hover:opacity-90 disabled:opacity-40 rounded-xl text-sm text-white transition-colors"
              >
                {site.name}
              </button>
            ))}
          </div>
        </div>

        <div className="bg-[var(--bg-card)] rounded-2xl p-5">
          <h3 className="font-medium mb-3">切换 Profile</h3>
          <div className="flex flex-wrap gap-2.5">
            {profiles.map(p => (
              <button
                key={p.name}
                onClick={() => handleSwitch('profile', p.name)}
                disabled={switching}
                className={`px-4 py-2.5 rounded-xl text-sm transition-all ${
                  routing?.active_profile === p.name
                    ? 'bg-[var(--accent-green-bg)] text-[var(--accent-green-text)] ring-1 ring-current/20'
                    : 'bg-[var(--bg-input)] hover:bg-[var(--bg-hover)] text-[var(--text-secondary)]'
                } disabled:opacity-40`}
              >
                {p.name} ({p.rules.length} 规则)
              </button>
            ))}
          </div>
        </div>

        <div className="bg-[var(--bg-card)] rounded-2xl p-5 space-y-5">
          <h3 className="font-medium">Profile 管理</h3>

          <div className="flex gap-3 items-end">
            <div className="flex-1">
              <Input label="新 Profile 名称" value={newProfileName} onChange={setNewProfileName} placeholder="production" />
            </div>
            <button onClick={handleCreateProfile} className="px-5 py-2.5 bg-green-500 hover:bg-green-400 rounded-xl text-sm text-white transition-colors">
              创建
            </button>
          </div>

          {profiles.map(p => (
            <div key={p.name} className="bg-[var(--bg-sidebar)] rounded-xl p-4 space-y-3">
              <div className="flex items-center justify-between">
                <span className="font-medium">{p.name}</span>
                <div className="flex gap-2">
                  {routing?.active_profile === p.name && (
                    <span className="text-xs px-2.5 py-1 bg-[var(--accent-green-bg)] text-[var(--accent-green-text)] rounded-lg">活跃</span>
                  )}
                  <button
                    onClick={() => handleDeleteProfile(p.name)}
                    className="text-xs px-2.5 py-1 bg-[var(--accent-red-bg)] text-[var(--accent-red-text)] hover:opacity-80 rounded-lg transition-colors"
                  >
                    删除
                  </button>
                </div>
              </div>

              {p.rules.map((rule: Rule, idx: number) => (
                <div key={idx} className="flex items-center gap-2 text-sm pl-4">
                  <span className="font-mono text-[var(--accent-blue-text)]">{rule.model_pattern}</span>
                  <span className="text-[var(--text-muted)]">{'→'}</span>
                  <span className="px-2.5 py-0.5 bg-[var(--bg-input)] rounded-lg">{rule.site}</span>
                  {rule.fallback && (
                    <>
                      <span className="text-[var(--text-muted)]">fallback:</span>
                      <span className="px-2.5 py-0.5 bg-[var(--bg-input)] rounded-lg">{rule.fallback}</span>
                    </>
                  )}
                  <button
                    onClick={() => handleDeleteRule(p.name, idx)}
                    className="text-xs text-[var(--accent-red-text)] opacity-60 hover:opacity-100 ml-2"
                  >
                    x
                  </button>
                </div>
              ))}

              <div className="flex gap-2 items-end pl-4 pt-3 border-t border-[var(--border-subtle)]">
                <select
                  value={selectedProfile}
                  onChange={e => setSelectedProfile(e.target.value)}
                  className="hidden"
                >
                  <option value={p.name}>{p.name}</option>
                </select>
                <Input label="模型" value={newRule.model_pattern} onChange={v => setNewRule({ ...newRule, model_pattern: v })} placeholder="gpt-*" />
                <Select
                  label="站点"
                  value={newRule.site}
                  onChange={v => setNewRule({ ...newRule, site: v })}
                  options={[{ value: '', label: '选择站点' }, ...sites.map(s => ({ value: s.id, label: s.name }))]}
                />
                <Input label="Fallback" value={newRule.fallback} onChange={v => setNewRule({ ...newRule, fallback: v })} placeholder="可选" />
                <button
                  onClick={() => { setSelectedProfile(p.name); setTimeout(handleAddRule, 0); }}
                  className="px-4 py-2.5 bg-[var(--accent-blue)] hover:opacity-90 rounded-xl text-sm text-white transition-colors shrink-0"
                >
                  +
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function SitesPage({ sites, onRefresh }: { sites: Site[]; onRefresh: () => void }) {
  const [showAdd, setShowAdd] = useState(false);
  const [editSite, setEditSite] = useState<Site | null>(null);
  const [form, setForm] = useState({ id: '', name: '', base_url: '', api_key: '', protocol: 'openai', models: '' });
  const [submitting, setSubmitting] = useState(false);
  const [msg, setMsg] = useState('');

  const resetForm = () => setForm({ id: '', name: '', base_url: '', api_key: '', protocol: 'openai', models: '' });

  const startAdd = () => {
    resetForm();
    setEditSite(null);
    setShowAdd(true);
  };

  const startEdit = (site: Site) => {
    setForm({
      id: site.id,
      name: site.name,
      base_url: site.base_url,
      api_key: site.api_key,
      protocol: site.protocol,
      models: site.models.join(', '),
    });
    setEditSite(site);
    setShowAdd(true);
  };

  const handleSubmit = async () => {
    try {
      setSubmitting(true);
      const siteData = {
        ...form,
        models: form.models.split(',').map(m => m.trim()).filter(Boolean),
      };

      if (editSite) {
        await api.updateSite(siteData as Site);
        setMsg('站点已更新');
      } else {
        await api.addSite(siteData as Site);
        setMsg('站点添加成功');
      }
      setShowAdd(false);
      resetForm();
      setEditSite(null);
      onRefresh();
    } catch (e: any) {
      setMsg(`${editSite ? '更新' : '添加'}失败: ${e.message}`);
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm('确定删除此站点？相关路由规则需手动清理。')) return;
    try {
      await api.deleteSite(id);
      setMsg('站点已删除');
      onRefresh();
    } catch (e: any) {
      setMsg(`删除失败: ${e.message}`);
    }
  };

  return (
    <div className="p-8 space-y-8">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-semibold tracking-tight">站点管理</h2>
        <button
          onClick={startAdd}
          className="px-5 py-2.5 bg-[var(--accent-blue)] hover:opacity-90 rounded-xl text-sm text-white transition-colors"
        >
          添加站点
        </button>
      </div>

      {msg && <div className="p-4 bg-[var(--accent-blue-bg)] rounded-2xl text-[var(--accent-blue-text)] text-sm">{msg}</div>}

      {showAdd && (
        <div className="bg-[var(--bg-card)] rounded-2xl p-5 space-y-4">
          <h3 className="font-medium">{editSite ? '编辑站点' : '添加新站点'}</h3>
          <div className="grid grid-cols-2 gap-4">
            {editSite ? (
              <Input label="Site ID" value={form.id} onChange={() => {}} disabled />
            ) : (
              <Input label="Site ID" value={form.id} onChange={v => setForm({ ...form, id: v })} placeholder="openai-official" />
            )}
            <Input label="名称" value={form.name} onChange={v => setForm({ ...form, name: v })} placeholder="OpenAI 官方" />
            <Input label="Base URL" value={form.base_url} onChange={v => setForm({ ...form, base_url: v })} placeholder="https://api.openai.com" />
            <Input label="API Key" value={form.api_key} onChange={v => setForm({ ...form, api_key: v })} placeholder={editSite ? '留空则不修改' : 'sk-...'} type="password" />
            <Select
              label="协议"
              value={form.protocol}
              onChange={v => setForm({ ...form, protocol: v })}
              options={[
                { value: 'openai', label: 'OpenAI' },
                { value: 'anthropic', label: 'Anthropic' },
                { value: 'gemini', label: 'Gemini' },
              ]}
            />
            <Input label="模型(逗号分隔)" value={form.models} onChange={v => setForm({ ...form, models: v })} placeholder="gpt-4o,gpt-4o-mini" />
          </div>
          <div className="flex gap-3">
            <button
              onClick={handleSubmit}
              disabled={submitting}
              className="px-5 py-2.5 bg-green-500 hover:bg-green-400 disabled:opacity-40 rounded-xl text-sm text-white transition-colors"
            >
              {submitting ? '处理中...' : editSite ? '保存修改' : '确认添加'}
            </button>
            <button
              onClick={() => { setShowAdd(false); setEditSite(null); resetForm(); }}
              className="px-5 py-2.5 bg-[var(--bg-input)] hover:bg-[var(--bg-hover)] rounded-xl text-sm transition-colors"
            >
              取消
            </button>
          </div>
        </div>
      )}

      <div className="space-y-4">
        {sites.map(site => (
          <div key={site.id} className="bg-[var(--bg-card)] rounded-2xl p-5 hover:bg-[var(--bg-hover)] transition-colors">
            <div className="flex items-center justify-between mb-3">
              <div>
                <span className="font-medium">{site.name}</span>
                <span className="ml-2 text-xs text-[var(--text-muted)]">{site.id}</span>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-xs px-2.5 py-1 bg-[var(--bg-input)] rounded-lg">{site.protocol}</span>
                <button
                  onClick={() => startEdit(site)}
                  className="text-xs px-2.5 py-1 bg-[var(--accent-blue-bg)] text-[var(--accent-blue-text)] hover:opacity-80 rounded-lg transition-colors"
                >
                  编辑
                </button>
                <button
                  onClick={() => handleDelete(site.id)}
                  className="text-xs px-2.5 py-1 bg-[var(--accent-red-bg)] text-[var(--accent-red-text)] hover:opacity-80 rounded-lg transition-colors"
                >
                  删除
                </button>
              </div>
            </div>
            <p className="text-xs text-[var(--text-muted)] mb-2">{site.base_url}</p>
            <p className="text-xs text-[var(--text-muted)] opacity-50 mb-3">Key: {site.api_key.slice(0, 8)}...{site.api_key.slice(-4)}</p>
            <div className="flex flex-wrap gap-1.5">
              {site.models.map((m: string) => (
                <span key={m} className="text-xs px-2 py-0.5 bg-[var(--accent-blue-bg)] text-[var(--accent-blue-text)] rounded-lg">{m}</span>
              ))}
            </div>
          </div>
        ))}
        {sites.length === 0 && (
          <div className="text-center py-10 text-[var(--text-muted)]">暂无站点，点击上方"添加站点"开始</div>
        )}
      </div>
    </div>
  );
}

function LogsPage() {
  const [logs, setLogs] = useState<RequestLog[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchLogs = async () => {
    try {
      setLoading(true);
      const data = await api.getLogs();
      setLogs(data);
    } catch { /* ignore */ } finally {
      setLoading(false);
    }
  };

  useEffect(() => { fetchLogs(); }, []);

  return (
    <div className="p-8 space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-semibold tracking-tight">请求日志</h2>
        <button onClick={fetchLogs} className="px-4 py-2 bg-[var(--bg-input)] hover:bg-[var(--bg-hover)] rounded-xl text-sm transition-colors">
          刷新
        </button>
      </div>

      <div className="bg-[var(--bg-card)] rounded-2xl overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-[var(--border-subtle)]">
              <th className="text-left px-5 py-3 text-[var(--text-muted)] font-medium">时间</th>
              <th className="text-left px-5 py-3 text-[var(--text-muted)] font-medium">站点</th>
              <th className="text-left px-5 py-3 text-[var(--text-muted)] font-medium">模型</th>
              <th className="text-left px-5 py-3 text-[var(--text-muted)] font-medium">耗时</th>
              <th className="text-left px-5 py-3 text-[var(--text-muted)] font-medium">Tokens</th>
              <th className="text-left px-5 py-3 text-[var(--text-muted)] font-medium">状态</th>
            </tr>
          </thead>
          <tbody>
            {logs.map(log => (
              <tr key={log.id} className="border-b border-[var(--border-subtle)] hover:bg-[var(--bg-hover)]">
                <td className="px-5 py-3 text-xs text-[var(--text-muted)]">{new Date(log.timestamp).toLocaleTimeString()}</td>
                <td className="px-5 py-3">{log.site_id}</td>
                <td className="px-5 py-3 font-mono text-[var(--accent-blue-text)] text-xs">{log.model}</td>
                <td className="px-5 py-3">{log.latency_ms}ms</td>
                <td className="px-5 py-3 text-xs">{log.input_tokens}/{log.output_tokens}</td>
                <td className="px-5 py-3">
                  <span className={`text-xs px-2 py-0.5 rounded-lg ${
                    log.status_code >= 400 ? 'bg-[var(--accent-red-bg)] text-[var(--accent-red-text)]' :
                    log.status_code >= 300 ? 'bg-[var(--accent-yellow-bg)] text-[var(--accent-yellow-text)]' :
                    'bg-[var(--accent-green-bg)] text-[var(--accent-green-text)]'
                  }`}>
                    {log.status_code}
                  </span>
                  {log.is_stream && <span className="ml-1.5 text-xs text-[var(--text-muted)]">stream</span>}
                </td>
              </tr>
            ))}
            {logs.length === 0 && !loading && (
              <tr><td colSpan={6} className="px-5 py-10 text-center text-[var(--text-muted)]">暂无日志</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function StatsPage() {
  const [stats, setStats] = useState<Stats | null>(null);
  const [days, setDays] = useState('1');

  useEffect(() => {
    api.getStats(Number(days)).then(setStats).catch(() => {});
  }, [days]);

  return (
    <div className="p-8 space-y-8">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-semibold tracking-tight">用量统计</h2>
        <Select
          value={days}
          onChange={setDays}
          options={[
            { value: '1', label: '今日' },
            { value: '7', label: '近7天' },
            { value: '30', label: '近30天' },
          ]}
        />
      </div>

      <div className="grid grid-cols-4 gap-5">
        <Card title="请求总数" value={String(stats?.total_requests || 0)} />
        <Card title="输入Tokens" value={formatNumber(stats?.total_input_tokens || 0)} />
        <Card title="输出Tokens" value={formatNumber(stats?.total_output_tokens || 0)} />
        <Card title="预估费用" value={formatCost(stats?.total_cost || 0)} />
      </div>
    </div>
  );
}

function SettingsPage({ theme, onThemeChange }: { theme: Theme; onThemeChange: (t: Theme) => void }) {
  const [config, setConfig] = useState<AppConfig | null>(null);
  const [msg, setMsg] = useState('');
  const [saving, setSaving] = useState(false);

  const [proxyListen, setProxyListen] = useState('');
  const [webListen, setWebListen] = useState('');
  const [accessToken, setAccessToken] = useState('');
  const [allowedIPs, setAllowedIPs] = useState('');
  const [globalRPM, setGlobalRPM] = useState(60);
  const [logEnabled, setLogEnabled] = useState(true);
  const [logMaxDays, setLogMaxDays] = useState(30);

  useEffect(() => {
    api.getConfig().then(c => {
      setConfig(c);
      setProxyListen(c.proxy.listen);
      setWebListen(c.proxy.web_listen);
      setAccessToken(c.security.access_token);
      setAllowedIPs(c.security.allowed_ips.join(', '));
      setGlobalRPM(c.security.rate_limit.global_rpm);
      setLogEnabled(c.logging.enabled);
      setLogMaxDays(c.logging.max_days);
    }).catch(() => {});
  }, []);

  const handleSave = async () => {
    try {
      setSaving(true);
      await api.updateConfig({
        proxy: { listen: proxyListen, web_listen: webListen },
        security: {
          api_key_encryption: config!.security.api_key_encryption,
          access_token: accessToken,
          allowed_ips: allowedIPs.split(',').map(s => s.trim()).filter(Boolean),
          rate_limit: { global_rpm: globalRPM, per_site_rpm: config!.security.rate_limit.per_site_rpm },
        },
        logging: { enabled: logEnabled, max_days: logMaxDays, log_body: config!.logging.log_body },
      });
      setMsg('配置已保存');
    } catch (e: any) {
      setMsg(`保存失败: ${e.message}`);
    } finally {
      setSaving(false);
    }
  };

  const handleReload = async () => {
    try {
      await api.reloadConfig();
      setMsg('配置已重新加载');
    } catch (e: any) {
      setMsg(`重载失败: ${e.message}`);
    }
  };

  if (!config) return <div className="p-8 text-[var(--text-muted)]">加载中...</div>;

  const themeOptions: { value: Theme; label: string }[] = [
    { value: 'light', label: '浅色' },
    { value: 'dark', label: '暗色' },
    { value: 'system', label: '跟随系统' },
  ];

  return (
    <div className="p-8 space-y-8">
      <h2 className="text-2xl font-semibold tracking-tight">设置</h2>

      {msg && <div className="p-4 bg-[var(--accent-blue-bg)] rounded-2xl text-[var(--accent-blue-text)] text-sm">{msg}</div>}

      <div className="space-y-5">
        <div className="bg-[var(--bg-card)] rounded-2xl p-5 space-y-4">
          <h3 className="font-medium">外观</h3>
          <div className="flex gap-2">
            {themeOptions.map(opt => (
              <button
                key={opt.value}
                onClick={() => onThemeChange(opt.value)}
                className={`px-4 py-2.5 rounded-xl text-sm transition-all ${
                  theme === opt.value
                    ? 'bg-[var(--accent-blue)] text-white'
                    : 'bg-[var(--bg-input)] text-[var(--text-secondary)] hover:bg-[var(--bg-hover)]'
                }`}
              >
                {opt.label}
              </button>
            ))}
          </div>
        </div>

        <div className="bg-[var(--bg-card)] rounded-2xl p-5 space-y-4">
          <h3 className="font-medium">代理设置</h3>
          <div className="grid grid-cols-2 gap-4">
            <Input label="代理监听地址" value={proxyListen} onChange={setProxyListen} placeholder="127.0.0.1:9090" />
            <Input label="Web UI 地址" value={webListen} onChange={setWebListen} placeholder="127.0.0.1:9091" />
          </div>
        </div>

        <div className="bg-[var(--bg-card)] rounded-2xl p-5 space-y-4">
          <h3 className="font-medium">安全设置</h3>
          <Input label="Access Token" value={accessToken} onChange={setAccessToken} placeholder="留空则不鉴权" type="password" />
          <Input label="IP 白名单(逗号分隔)" value={allowedIPs} onChange={setAllowedIPs} placeholder="127.0.0.1, 10.0.0.1" />
          <Input label="全局 RPM 限制" value={String(globalRPM)} onChange={v => setGlobalRPM(Number(v) || 0)} placeholder="60" />
        </div>

        <div className="bg-[var(--bg-card)] rounded-2xl p-5 space-y-4">
          <h3 className="font-medium">日志设置</h3>
          <div className="flex items-center gap-3">
            <label className="text-sm text-[var(--text-muted)]">启用日志</label>
            <button
              onClick={() => setLogEnabled(!logEnabled)}
              className={`w-11 h-6 rounded-full transition-colors ${logEnabled ? 'bg-[var(--toggle-active)]' : 'bg-[var(--toggle-bg)]'}`}
            >
              <div className={`w-5 h-5 rounded-full bg-[var(--toggle-knob)] shadow-sm transition-transform ${logEnabled ? 'translate-x-5.5' : 'translate-x-0.5'}`} />
            </button>
          </div>
          <Input label="日志保留天数" value={String(logMaxDays)} onChange={v => setLogMaxDays(Number(v) || 0)} placeholder="30" />
        </div>

        <div className="flex gap-3">
          <button
            onClick={handleSave}
            disabled={saving}
            className="px-5 py-2.5 bg-[var(--accent-blue)] hover:opacity-90 disabled:opacity-40 rounded-xl text-sm text-white transition-colors"
          >
            {saving ? '保存中...' : '保存配置'}
          </button>
          <button
            onClick={handleReload}
            className="px-5 py-2.5 bg-[var(--bg-input)] hover:bg-[var(--bg-hover)] rounded-xl text-sm transition-colors"
          >
            重新加载配置
          </button>
        </div>
      </div>
    </div>
  );
}

function Card({ title, value }: { title: string; value: string }) {
  return (
    <div className="bg-[var(--bg-card)] rounded-2xl p-5">
      <p className="text-xs text-[var(--text-muted)] mb-2">{title}</p>
      <p className="text-2xl font-semibold">{value}</p>
    </div>
  );
}

function Input({ label, value, onChange, placeholder, type = 'text', disabled = false }: {
  label: string; value: string; onChange: (v: string) => void; placeholder?: string; type?: string; disabled?: boolean;
}) {
  return (
    <div>
      <label className="block text-xs text-[var(--text-muted)] mb-1.5">{label}</label>
      <input
        type={type}
        value={value}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        disabled={disabled}
        className={`w-full px-3.5 py-2.5 bg-[var(--bg-input)] rounded-xl text-sm focus:outline-none focus:ring-1 focus:ring-[var(--accent-blue)]/50 placeholder:text-[var(--text-muted)] ${disabled ? 'opacity-40 cursor-not-allowed' : ''}`}
      />
    </div>
  );
}

function Select({ label, value, onChange, options }: {
  label?: string; value: string; onChange: (v: string) => void; options: { value: string; label: string }[];
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, [open]);

  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false);
    };
    document.addEventListener('keydown', handler);
    return () => document.removeEventListener('keydown', handler);
  }, []);

  const selected = options.find(o => o.value === value);

  return (
    <div className="relative" ref={ref}>
      {label && <label className="block text-xs text-[var(--text-muted)] mb-1.5">{label}</label>}
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="w-full px-3.5 py-2.5 bg-[var(--bg-input)] rounded-xl text-sm text-left focus:outline-none focus:ring-1 focus:ring-[var(--accent-blue)]/50 flex items-center justify-between"
      >
        <span className={!selected?.value ? 'text-[var(--text-muted)]' : ''}>{selected?.label || '请选择'}</span>
        <svg className={`w-4 h-4 text-[var(--text-muted)] transition-transform ${open ? 'rotate-180' : ''}`} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
        </svg>
      </button>
      {open && (
        <div className="absolute z-50 mt-1 w-full bg-[var(--select-dropdown)] rounded-xl shadow-lg border border-[var(--border-subtle)] py-1 max-h-60 overflow-auto">
          {options.map(opt => (
            <button
              key={opt.value}
              type="button"
              onClick={() => { onChange(opt.value); setOpen(false); }}
              className={`w-full text-left px-3.5 py-2.5 text-sm transition-colors ${
                opt.value === value
                  ? 'bg-[var(--accent-blue-bg)] text-[var(--accent-blue-text)]'
                  : 'hover:bg-[var(--select-hover)] text-[var(--text-primary)]'
              }`}
            >
              {opt.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

function formatNumber(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M';
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K';
  return String(n);
}

function formatCost(c: number): string {
  if (c < 0.01) return '$' + c.toFixed(4);
  return '$' + c.toFixed(2);
}
