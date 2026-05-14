import { useState, useEffect } from 'react';
import { api } from './api/client';
import type { RoutingCurrent, Site, RequestLog, Stats } from './api/client';

type Page = 'dashboard' | 'switch' | 'sites' | 'logs' | 'stats' | 'settings';

export default function App() {
  const [page, setPage] = useState<Page>('dashboard');
  const [routing, setRouting] = useState<RoutingCurrent | null>(null);
  const [sites, setSites] = useState<Site[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  const refresh = async () => {
    try {
      setLoading(true);
      setError('');
      const [r, s] = await Promise.all([api.getRoutingCurrent(), api.getSites()]);
      setRouting(r);
      setSites(s);
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
    <div className="flex h-screen bg-gray-950 text-gray-100">
      <nav className="w-48 bg-gray-900 border-r border-gray-800 flex flex-col">
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
            {page === 'switch' && <SwitchPage routing={routing} sites={sites} onSwitch={refresh} />}
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
                {site.models.map(m => (
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
              {routing?.profile?.rules?.map((rule, i) => (
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

function SwitchPage({ routing, sites, onSwitch }: { routing: RoutingCurrent | null; sites: Site[]; onSwitch: () => void }) {
  const [switching, setSwitching] = useState(false);
  const [msg, setMsg] = useState('');

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
          <h3 className="font-semibold mb-3">当前路由规则</h3>
          <div className="space-y-2">
            {routing?.profile?.rules?.map((rule, i) => (
              <div key={i} className="flex items-center gap-3 text-sm">
                <span className="font-mono text-blue-400 w-32">{rule.model_pattern}</span>
                <span className="text-gray-500">{'→'}</span>
                <span className="px-2 py-0.5 bg-gray-800 rounded">{rule.site}</span>
                {rule.fallback && (
                  <>
                    <span className="text-gray-500">fallback:</span>
                    <span className="px-2 py-0.5 bg-gray-800 rounded">{rule.fallback}</span>
                  </>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

function SitesPage({ sites, onRefresh }: { sites: Site[]; onRefresh: () => void }) {
  const [showAdd, setShowAdd] = useState(false);
  const [newSite, setNewSite] = useState({ id: '', name: '', base_url: '', api_key: '', protocol: 'openai', models: '' });
  const [adding, setAdding] = useState(false);
  const [msg, setMsg] = useState('');

  const handleAdd = async () => {
    try {
      setAdding(true);
      await api.addSite({
        ...newSite,
        models: newSite.models.split(',').map(m => m.trim()).filter(Boolean),
      });
      setMsg('站点添加成功');
      setShowAdd(false);
      setNewSite({ id: '', name: '', base_url: '', api_key: '', protocol: 'openai', models: '' });
      onRefresh();
    } catch (e: any) {
      setMsg(`添加失败: ${e.message}`);
    } finally {
      setAdding(false);
    }
  };

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold">站点管理</h2>
        <button
          onClick={() => setShowAdd(!showAdd)}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-500 rounded text-sm transition-colors"
        >
          {showAdd ? '取消' : '添加站点'}
        </button>
      </div>

      {msg && <div className="p-3 bg-blue-900/20 border border-blue-800 rounded text-blue-400 text-sm">{msg}</div>}

      {showAdd && (
        <div className="bg-gray-900 border border-gray-800 rounded-lg p-4 space-y-3">
          <h3 className="font-semibold">添加新站点</h3>
          <div className="grid grid-cols-2 gap-3">
            <Input label="Site ID" value={newSite.id} onChange={v => setNewSite({ ...newSite, id: v })} placeholder="openai-official" />
            <Input label="名称" value={newSite.name} onChange={v => setNewSite({ ...newSite, name: v })} placeholder="OpenAI 官方" />
            <Input label="Base URL" value={newSite.base_url} onChange={v => setNewSite({ ...newSite, base_url: v })} placeholder="https://api.openai.com" />
            <Input label="API Key" value={newSite.api_key} onChange={v => setNewSite({ ...newSite, api_key: v })} placeholder="sk-..." type="password" />
            <div>
              <label className="block text-xs text-gray-400 mb-1">协议</label>
              <select
                value={newSite.protocol}
                onChange={e => setNewSite({ ...newSite, protocol: e.target.value })}
                className="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded text-sm"
              >
                <option value="openai">OpenAI</option>
                <option value="anthropic">Anthropic</option>
                <option value="gemini">Gemini</option>
              </select>
            </div>
            <Input label="模型(逗号分隔)" value={newSite.models} onChange={v => setNewSite({ ...newSite, models: v })} placeholder="gpt-4o,gpt-4o-mini" />
          </div>
          <button
            onClick={handleAdd}
            disabled={adding}
            className="px-4 py-2 bg-green-600 hover:bg-green-500 disabled:opacity-50 rounded text-sm transition-colors"
          >
            {adding ? '添加中...' : '确认添加'}
          </button>
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
              <span className="text-xs px-2 py-0.5 bg-gray-800 rounded">{site.protocol}</span>
            </div>
            <p className="text-xs text-gray-500 mb-2">{site.base_url}</p>
            <div className="flex flex-wrap gap-1">
              {site.models.map(m => (
                <span key={m} className="text-xs px-1.5 py-0.5 bg-blue-900/30 text-blue-400 rounded">{m}</span>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function LogsPage() {
  const [logs, setLogs] = useState<RequestLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter] = useState('');

  const fetchLogs = async () => {
    try {
      setLoading(true);
      const data = await api.getLogs(filter);
      setLogs(data);
    } catch { } finally {
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
  const [msg, setMsg] = useState('');

  const handleReload = async () => {
    try {
      await api.reloadConfig();
      setMsg('配置已重新加载');
    } catch (e: any) {
      setMsg(`重载失败: ${e.message}`);
    }
  };

  return (
    <div className="p-6 space-y-6">
      <h2 className="text-2xl font-bold">设置</h2>

      <div className="bg-gray-900 border border-gray-800 rounded-lg p-4 space-y-4">
        <h3 className="font-semibold">配置管理</h3>
        <p className="text-sm text-gray-400">配置文件位于 ~/.mswitch/config.yaml，修改后点击重载生效。</p>
        <button
          onClick={handleReload}
          className="px-4 py-2 bg-blue-600 hover:bg-blue-500 rounded text-sm transition-colors"
        >
          重新加载配置
        </button>
        {msg && <p className="text-sm text-blue-400">{msg}</p>}
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

function Input({ label, value, onChange, placeholder, type = 'text' }: {
  label: string; value: string; onChange: (v: string) => void; placeholder?: string; type?: string;
}) {
  return (
    <div>
      <label className="block text-xs text-gray-400 mb-1">{label}</label>
      <input
        type={type}
        value={value}
        onChange={e => onChange(e.target.value)}
        placeholder={placeholder}
        className="w-full px-3 py-2 bg-gray-800 border border-gray-700 rounded text-sm focus:outline-none focus:border-blue-500"
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
