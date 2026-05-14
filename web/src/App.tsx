import { useState, useEffect } from 'react';
import { api } from './api/client';
import type { RoutingCurrent, Site, Profile, Rule, Stats, RequestLog, AppConfig } from './api/client';

type Page = 'dashboard' | 'switch' | 'sites' | 'logs' | 'stats' | 'settings';

export default function App() {
  const [page, setPage] = useState<Page>('dashboard');
  const [routing, setRouting] = useState<RoutingCurrent | null>(null);
  const [sites, setSites] = useState<Site[]>([]);
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

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

  if (!loading && sites.length === 0) {
    return <Onboarding onComplete={refresh} />;
  }

  return (
    <div className="flex h-screen bg-gray-950 text-gray-100">
      <nav className="w-48 bg-gray-900 border-r border-gray-800 flex flex-col shrink-0">
        <div className="p-4 border-b border-gray-800">
          <h1 className="text-lg font-bold text-white">mswitch</h1>
          <p className="text-xs text-gray-500 mt-1">
            {routing ? `Profile: ${routing.active_profile}` : 'loading...'}
          </p>
        </div>
        <div className="flex-1 py-2">
          {navItems.map(item => (
            <button
              key={item.key}
              onClick={() => setPage(item.key)}
              className={`w-full text-left px-4 py-2.5 text-sm transition-colors ${
                page === item.key
                  ? 'bg-blue-600/20 text-blue-400 border-r-2 border-blue-400'
                  : 'text-gray-400 hover:text-gray-200 hover:bg-gray-800'
              }`}
            >
              {item.label}
            </button>
          ))}
        </div>
        <div className="p-4 border-t border-gray-800">
          <button
            onClick={refresh}
            className="w-full px-3 py-1.5 text-xs bg-gray-800 hover:bg-gray-700 rounded transition-colors"
          >
            刷新
          </button>
        </div>
      </nav>

      <main className="flex-1 overflow-auto">
        {error && (
          <div className="m-4 p-3 bg-red-900/30 border border-red-800 rounded text-red-400 text-sm">
            {error}
          </div>
        )}

        {loading && !routing ? (
          <div className="flex items-center justify-center h-full text-gray-500">加载中...</div>
        ) : (
          <>
            {page === 'dashboard' && <DashboardPage routing={routing} sites={sites} />}
            {page === 'switch' && <SwitchPage routing={routing} sites={sites} profiles={profiles} onSwitch={refresh} />}
            {page === 'sites' && <SitesPage sites={sites} onRefresh={refresh} />}
            {page === 'logs' && <LogsPage />}
            {page === 'stats' && <StatsPage />}
            {page === 'settings' && <SettingsPage />}
          </>
        )}
      </main>
    </div>
  );
}

function Onboarding({ onComplete }: { onComplete: () => void }) {
  const [form, setForm] = useState({ id: '', name: '', base_url: '', api_key: '', protocol: 'openai', models: '' });
  const [adding, setAdding] = useState(false);
  const [msg, setMsg] = useState('');

  const handleSubmit = async () => {
    try {
      setAdding(true);
      await api.addSite({
        ...form,
        models: form.models.split(',').map(m => m.trim()).filter(Boolean),
      });
      setMsg('站点添加成功！');
      setTimeout(onComplete, 500);
    } catch (e: any) {
      setMsg(`添加失败: ${e.message}`);
    } finally {
      setAdding(false);
    }
  };

  return (
    <div className="flex items-center justify-center min-h-screen bg-gray-950 text-gray-100">
      <div className="w-full max-w-lg p-8">
        <div className="text-center mb-8">
          <h1 className="text-4xl font-bold mb-2">mswitch</h1>
          <p className="text-gray-400">Model API Hot-Switch Proxy</p>
        </div>

        <div className="bg-gray-900 border border-gray-800 rounded-lg p-6 space-y-4">
          <h2 className="text-xl font-semibold text-center">添加你的第一个 API 站点</h2>
          <p className="text-sm text-gray-400 text-center">配置完成后即可开始使用 mswitch</p>

          <div className="space-y-3">
            <Input label="Site ID" value={form.id} onChange={v => setForm({ ...form, id: v })} placeholder="openai-official" />
            <Input label="名称" value={form.name} onChange={v => setForm({ ...form, name: v })} placeholder="OpenAI 官方" />
            <Input label="Base URL" value={form.base_url} onChange={v => setForm({ ...form, base_url: v })} placeholder="https://api.openai.com" />
            <Input label="API Key" value={form.api_key} onChange={v => setForm({ ...form, api_key: v })} placeholder="sk-..." type="password" />
            <div>
              <label className="block text-xs text-gray-400 mb-1">协议</label>
              <select
                value={form.protocol}
                onChange={e => setForm({ ...form, protocol: e.target.value })}
                className="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded text-sm"
              >
                <option value="openai">OpenAI</option>
                <option value="anthropic">Anthropic</option>
                <option value="gemini">Gemini</option>
              </select>
            </div>
            <Input label="模型(逗号分隔)" value={form.models} onChange={v => setForm({ ...form, models: v })} placeholder="gpt-4o,gpt-4o-mini" />
          </div>

          <button
            onClick={handleSubmit}
            disabled={adding}
            className="w-full py-2.5 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 rounded text-sm font-medium transition-colors"
          >
            {adding ? '添加中...' : '开始使用'}
          </button>

          {msg && <p className="text-sm text-center text-blue-400">{msg}</p>}
        </div>
      </div>
    </div>
  );
}

function DashboardPage({ routing, sites }: { routing: RoutingCurrent | null; sites: Site[] }) {
  return (
    <div className="p-6 space-y-6">
      <h2 className="text-2xl font-bold">仪表盘</h2>

      <div className="grid grid-cols-3 gap-4">
        <Card title="站点数" value={String(sites.length)} />
        <Card title="活跃Profile" value={routing?.active_profile || '-'} />
        <Card title="路由规则" value={String(routing?.profile?.rules?.length || 0)} />
      </div>

      <div>
        <h3 className="text-lg font-semibold mb-3">站点列表</h3>
        <div className="grid grid-cols-2 gap-3">
          {sites.map(site => (
            <div key={site.id} className="bg-gray-900 border border-gray-800 rounded-lg p-4">
              <div className="flex items-center justify-between mb-2">
                <span className="font-medium">{site.name}</span>
                <span className="text-xs px-2 py-0.5 bg-gray-800 rounded">{site.protocol}</span>
              </div>
              <p className="text-xs text-gray-500 mb-2">{site.base_url}</p>
              <div className="flex flex-wrap gap-1">
                {site.models.map((m: string) => (
                  <span key={m} className="text-xs px-1.5 py-0.5 bg-blue-900/30 text-blue-400 rounded">
                    {m}
                  </span>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>

      <div>
        <h3 className="text-lg font-semibold mb-3">当前路由</h3>
        <div className="bg-gray-900 border border-gray-800 rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-gray-800">
                <th className="text-left px-4 py-2 text-gray-400 font-medium">模型匹配</th>
                <th className="text-left px-4 py-2 text-gray-400 font-medium">目标站点</th>
                <th className="text-left px-4 py-2 text-gray-400 font-medium">Fallback</th>
              </tr>
            </thead>
            <tbody>
              {routing?.profile?.rules?.map((rule: Rule, i: number) => (
                <tr key={i} className="border-b border-gray-800/50">
                  <td className="px-4 py-2 font-mono text-blue-400">{rule.model_pattern}</td>
                  <td className="px-4 py-2">{rule.site}</td>
                  <td className="px-4 py-2 text-gray-500">{rule.fallback || '-'}</td>
                </tr>
              ))}
              {(!routing?.profile?.rules?.length) && (
                <tr><td colSpan={3} className="px-4 py-4 text-center text-gray-500">无路由规则</td></tr>
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
    <div className="p-6 space-y-6">
      <h2 className="text-2xl font-bold">切换中心</h2>

      {msg && (
        <div className="p-3 bg-blue-900/20 border border-blue-800 rounded text-blue-400 text-sm">{msg}</div>
      )}

      <div className="space-y-4">
        <div className="bg-gray-900 border border-gray-800 rounded-lg p-4">
          <h3 className="font-semibold mb-3">快速切换站点</h3>
          <p className="text-sm text-gray-400 mb-3">将所有请求路由到指定站点</p>
          <div className="flex flex-wrap gap-2">
            {sites.map(site => (
              <button
                key={site.id}
                onClick={() => handleSwitch('site', site.id)}
                disabled={switching}
                className="px-4 py-2 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 rounded text-sm transition-colors"
              >
                {site.name}
              </button>
            ))}
          </div>
        </div>

        <div className="bg-gray-900 border border-gray-800 rounded-lg p-4">
          <h3 className="font-semibold mb-3">切换 Profile</h3>
          <div className="flex flex-wrap gap-2">
            {profiles.map(p => (
              <button
                key={p.name}
                onClick={() => handleSwitch('profile', p.name)}
                disabled={switching}
                className={`px-4 py-2 rounded text-sm transition-colors ${
                  routing?.active_profile === p.name
                    ? 'bg-green-600/30 text-green-400 border border-green-600'
                    : 'bg-gray-800 hover:bg-gray-700'
                } disabled:opacity-50`}
              >
                {p.name} ({p.rules.length} 规则)
              </button>
            ))}
          </div>
        </div>

        <div className="bg-gray-900 border border-gray-800 rounded-lg p-4 space-y-4">
          <h3 className="font-semibold">Profile 管理</h3>

          <div className="flex gap-2 items-end">
            <div className="flex-1">
              <Input label="新 Profile 名称" value={newProfileName} onChange={setNewProfileName} placeholder="production" />
            </div>
            <button onClick={handleCreateProfile} className="px-4 py-2 bg-green-600 hover:bg-green-500 rounded text-sm transition-colors">
              创建
            </button>
          </div>

          {profiles.map(p => (
            <div key={p.name} className="border border-gray-800 rounded p-3 space-y-2">
              <div className="flex items-center justify-between">
                <span className="font-medium">{p.name}</span>
                <div className="flex gap-2">
                  {routing?.active_profile === p.name && (
                    <span className="text-xs px-2 py-0.5 bg-green-900/30 text-green-400 rounded">活跃</span>
                  )}
                  <button
                    onClick={() => handleDeleteProfile(p.name)}
                    className="text-xs px-2 py-1 bg-red-900/30 text-red-400 hover:bg-red-900/50 rounded transition-colors"
                  >
                    删除
                  </button>
                </div>
              </div>

              {p.rules.map((rule: Rule, idx: number) => (
                <div key={idx} className="flex items-center gap-2 text-sm pl-4">
                  <span className="font-mono text-blue-400">{rule.model_pattern}</span>
                  <span className="text-gray-500">{'→'}</span>
                  <span className="px-2 py-0.5 bg-gray-800 rounded">{rule.site}</span>
                  {rule.fallback && (
                    <>
                      <span className="text-gray-500">fallback:</span>
                      <span className="px-2 py-0.5 bg-gray-800 rounded">{rule.fallback}</span>
                    </>
                  )}
                  <button
                    onClick={() => handleDeleteRule(p.name, idx)}
                    className="text-xs text-red-400 hover:text-red-300 ml-2"
                  >
                    x
                  </button>
                </div>
              ))}

              <div className="flex gap-2 items-end pl-4 pt-2 border-t border-gray-800/50">
                <select
                  value={selectedProfile}
                  onChange={e => setSelectedProfile(e.target.value)}
                  className="hidden"
                >
                  <option value={p.name}>{p.name}</option>
                </select>
                <Input label="模型" value={newRule.model_pattern} onChange={v => setNewRule({ ...newRule, model_pattern: v })} placeholder="gpt-*" />
                <select
                  value={newRule.site}
                  onChange={e => setNewRule({ ...newRule, site: e.target.value })}
                  className="px-3 py-2 bg-gray-800 border border-gray-700 rounded text-sm"
                >
                  <option value="">选择站点</option>
                  {sites.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
                </select>
                <Input label="Fallback" value={newRule.fallback} onChange={v => setNewRule({ ...newRule, fallback: v })} placeholder="可选" />
                <button
                  onClick={() => { setSelectedProfile(p.name); setTimeout(handleAddRule, 0); }}
                  className="px-3 py-2 bg-blue-600 hover:bg-blue-500 rounded text-sm transition-colors shrink-0"
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
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">站点管理</h2>
        <button
          onClick={startAdd}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-500 rounded text-sm transition-colors"
        >
          添加站点
        </button>
      </div>

      {msg && <div className="p-3 bg-blue-900/20 border border-blue-800 rounded text-blue-400 text-sm">{msg}</div>}

      {showAdd && (
        <div className="bg-gray-900 border border-gray-800 rounded-lg p-4 space-y-3">
          <h3 className="font-semibold">{editSite ? '编辑站点' : '添加新站点'}</h3>
          <div className="grid grid-cols-2 gap-3">
            {editSite ? (
              <Input label="Site ID" value={form.id} onChange={() => {}} disabled />
            ) : (
              <Input label="Site ID" value={form.id} onChange={v => setForm({ ...form, id: v })} placeholder="openai-official" />
            )}
            <Input label="名称" value={form.name} onChange={v => setForm({ ...form, name: v })} placeholder="OpenAI 官方" />
            <Input label="Base URL" value={form.base_url} onChange={v => setForm({ ...form, base_url: v })} placeholder="https://api.openai.com" />
            <Input label="API Key" value={form.api_key} onChange={v => setForm({ ...form, api_key: v })} placeholder={editSite ? '留空则不修改' : 'sk-...'} type="password" />
            <div>
              <label className="block text-xs text-gray-400 mb-1">协议</label>
              <select
                value={form.protocol}
                onChange={e => setForm({ ...form, protocol: e.target.value })}
                className="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded text-sm"
              >
                <option value="openai">OpenAI</option>
                <option value="anthropic">Anthropic</option>
                <option value="gemini">Gemini</option>
              </select>
            </div>
            <Input label="模型(逗号分隔)" value={form.models} onChange={v => setForm({ ...form, models: v })} placeholder="gpt-4o,gpt-4o-mini" />
          </div>
          <div className="flex gap-2">
            <button
              onClick={handleSubmit}
              disabled={submitting}
              className="px-4 py-2 bg-green-600 hover:bg-green-500 disabled:opacity-50 rounded text-sm transition-colors"
            >
              {submitting ? '处理中...' : editSite ? '保存修改' : '确认添加'}
            </button>
            <button
              onClick={() => { setShowAdd(false); setEditSite(null); resetForm(); }}
              className="px-4 py-2 bg-gray-800 hover:bg-gray-700 rounded text-sm transition-colors"
            >
              取消
            </button>
          </div>
        </div>
      )}

      <div className="space-y-3">
        {sites.map(site => (
          <div key={site.id} className="bg-gray-900 border border-gray-800 rounded-lg p-4">
            <div className="flex items-center justify-between mb-2">
              <div>
                <span className="font-medium">{site.name}</span>
                <span className="ml-2 text-xs text-gray-500">{site.id}</span>
              </div>
              <div className="flex items-center gap-2">
                <span className="text-xs px-2 py-0.5 bg-gray-800 rounded">{site.protocol}</span>
                <button
                  onClick={() => startEdit(site)}
                  className="text-xs px-2 py-1 bg-blue-900/30 text-blue-400 hover:bg-blue-900/50 rounded transition-colors"
                >
                  编辑
                </button>
                <button
                  onClick={() => handleDelete(site.id)}
                  className="text-xs px-2 py-1 bg-red-900/30 text-red-400 hover:bg-red-900/50 rounded transition-colors"
                >
                  删除
                </button>
              </div>
            </div>
            <p className="text-xs text-gray-500 mb-1">{site.base_url}</p>
            <p className="text-xs text-gray-600 mb-2">Key: {site.api_key.slice(0, 8)}...{site.api_key.slice(-4)}</p>
            <div className="flex flex-wrap gap-1">
              {site.models.map((m: string) => (
                <span key={m} className="text-xs px-1.5 py-0.5 bg-blue-900/30 text-blue-400 rounded">{m}</span>
              ))}
            </div>
          </div>
        ))}
        {sites.length === 0 && (
          <div className="text-center py-8 text-gray-500">暂无站点，点击上方"添加站点"开始</div>
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
    <div className="p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">请求日志</h2>
        <button onClick={fetchLogs} className="px-3 py-1.5 bg-gray-800 hover:bg-gray-700 rounded text-sm transition-colors">
          刷新
        </button>
      </div>

      <div className="bg-gray-900 border border-gray-800 rounded-lg overflow-hidden">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-gray-800">
              <th className="text-left px-3 py-2 text-gray-400 font-medium">时间</th>
              <th className="text-left px-3 py-2 text-gray-400 font-medium">站点</th>
              <th className="text-left px-3 py-2 text-gray-400 font-medium">模型</th>
              <th className="text-left px-3 py-2 text-gray-400 font-medium">耗时</th>
              <th className="text-left px-3 py-2 text-gray-400 font-medium">Tokens</th>
              <th className="text-left px-3 py-2 text-gray-400 font-medium">状态</th>
            </tr>
          </thead>
          <tbody>
            {logs.map(log => (
              <tr key={log.id} className="border-b border-gray-800/50 hover:bg-gray-800/30">
                <td className="px-3 py-2 text-xs text-gray-400">{new Date(log.timestamp).toLocaleTimeString()}</td>
                <td className="px-3 py-2">{log.site_id}</td>
                <td className="px-3 py-2 font-mono text-blue-400 text-xs">{log.model}</td>
                <td className="px-3 py-2">{log.latency_ms}ms</td>
                <td className="px-3 py-2 text-xs">{log.input_tokens}/{log.output_tokens}</td>
                <td className="px-3 py-2">
                  <span className={`text-xs px-1.5 py-0.5 rounded ${
                    log.status_code >= 400 ? 'bg-red-900/30 text-red-400' :
                    log.status_code >= 300 ? 'bg-yellow-900/30 text-yellow-400' :
                    'bg-green-900/30 text-green-400'
                  }`}>
                    {log.status_code}
                  </span>
                  {log.is_stream && <span className="ml-1 text-xs text-gray-500">stream</span>}
                </td>
              </tr>
            ))}
            {logs.length === 0 && !loading && (
              <tr><td colSpan={6} className="px-3 py-6 text-center text-gray-500">暂无日志</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function StatsPage() {
  const [stats, setStats] = useState<Stats | null>(null);
  const [days, setDays] = useState(1);

  useEffect(() => {
    api.getStats(days).then(setStats).catch(() => {});
  }, [days]);

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">用量统计</h2>
        <select
          value={days}
          onChange={e => setDays(Number(e.target.value))}
          className="px-3 py-1.5 bg-gray-800 border border-gray-700 rounded text-sm"
        >
          <option value={1}>今日</option>
          <option value={7}>近7天</option>
          <option value={30}>近30天</option>
        </select>
      </div>

      <div className="grid grid-cols-4 gap-4">
        <Card title="请求总数" value={String(stats?.total_requests || 0)} />
        <Card title="输入Tokens" value={formatNumber(stats?.total_input_tokens || 0)} />
        <Card title="输出Tokens" value={formatNumber(stats?.total_output_tokens || 0)} />
        <Card title="预估费用" value={formatCost(stats?.total_cost || 0)} />
      </div>
    </div>
  );
}

function SettingsPage() {
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

  if (!config) return <div className="p-6 text-gray-500">加载中...</div>;

  return (
    <div className="p-6 space-y-6">
      <h2 className="text-2xl font-bold">设置</h2>

      {msg && <div className="p-3 bg-blue-900/20 border border-blue-800 rounded text-blue-400 text-sm">{msg}</div>}

      <div className="space-y-4">
        <div className="bg-gray-900 border border-gray-800 rounded-lg p-4 space-y-3">
          <h3 className="font-semibold">代理设置</h3>
          <div className="grid grid-cols-2 gap-3">
            <Input label="代理监听地址" value={proxyListen} onChange={setProxyListen} placeholder="127.0.0.1:9090" />
            <Input label="Web UI 地址" value={webListen} onChange={setWebListen} placeholder="127.0.0.1:9091" />
          </div>
        </div>

        <div className="bg-gray-900 border border-gray-800 rounded-lg p-4 space-y-3">
          <h3 className="font-semibold">安全设置</h3>
          <Input label="Access Token" value={accessToken} onChange={setAccessToken} placeholder="留空则不鉴权" type="password" />
          <Input label="IP 白名单(逗号分隔)" value={allowedIPs} onChange={setAllowedIPs} placeholder="127.0.0.1, 10.0.0.1" />
          <Input label="全局 RPM 限制" value={String(globalRPM)} onChange={v => setGlobalRPM(Number(v) || 0)} placeholder="60" />
        </div>

        <div className="bg-gray-900 border border-gray-800 rounded-lg p-4 space-y-3">
          <h3 className="font-semibold">日志设置</h3>
          <div className="flex items-center gap-3">
            <label className="text-sm text-gray-400">启用日志</label>
            <button
              onClick={() => setLogEnabled(!logEnabled)}
              className={`w-10 h-5 rounded-full transition-colors ${logEnabled ? 'bg-blue-600' : 'bg-gray-700'}`}
            >
              <div className={`w-4 h-4 rounded-full bg-white transition-transform ${logEnabled ? 'translate-x-5' : 'translate-x-0.5'}`} />
            </button>
          </div>
          <Input label="日志保留天数" value={String(logMaxDays)} onChange={v => setLogMaxDays(Number(v) || 0)} placeholder="30" />
        </div>

        <div className="flex gap-3">
          <button
            onClick={handleSave}
            disabled={saving}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-500 disabled:opacity-50 rounded text-sm transition-colors"
          >
            {saving ? '保存中...' : '保存配置'}
          </button>
          <button
            onClick={handleReload}
            className="px-4 py-2 bg-gray-800 hover:bg-gray-700 rounded text-sm transition-colors"
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
    <div className="bg-gray-900 border border-gray-800 rounded-lg p-4">
      <p className="text-xs text-gray-400 mb-1">{title}</p>
      <p className="text-2xl font-bold">{value}</p>
    </div>
  );
}

function Input({ label, value, onChange, placeholder, type = 'text', disabled = false }: {
  label: string; value: string; onChange: (v: string) => void; placeholder?: string; type?: string; disabled?: boolean;
}) {
  return (
    <div>
      <label className="block text-xs text-gray-400 mb-1">{label}</label>
      <input
        type={type}
        value={value}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        disabled={disabled}
        className={`w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded text-sm focus:outline-none focus:border-blue-500 ${disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
      />
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
