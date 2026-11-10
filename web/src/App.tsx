import React, { useEffect, useState } from 'react';
import { Header } from './components/Header';
import { MetricCards } from './components/MetricCards';
import { WaterfallChart } from './components/WaterfallChart';
import { SecretShieldView } from './components/SecretShieldView';
import { StagedDiffView } from './components/StagedDiffView';
import { ConfigEditor } from './components/ConfigEditor';
import { StatusResponse, RunSummary, SecretFinding, DiffResponse, ConfigResponse } from './types';

export const App: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'overview' | 'secrets' | 'diff' | 'config'>('overview');
  const [status, setStatus] = useState<StatusResponse | null>(null);
  const [history, setHistory] = useState<RunSummary[]>([]);
  const [findings, setFindings] = useState<SecretFinding[]>([]);
  const [diffData, setDiffData] = useState<DiffResponse | null>(null);
  const [configData, setConfigData] = useState<ConfigResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [runningHook, setRunningHook] = useState(false);
  const [staging, setStaging] = useState(false);
  const [clearingCache, setClearingCache] = useState(false);

  const fetchData = async () => {
    setLoading(true);
    try {
      const [resStatus, resHistory, resDiff, resConfig, resScan] = await Promise.all([
        fetch('/api/status').then((r) => r.json()).catch(() => null),
        fetch('/api/history').then((r) => r.json()).catch(() => []),
        fetch('/api/diff').then((r) => r.json()).catch(() => null),
        fetch('/api/config').then((r) => r.json()).catch(() => null),
        fetch('/api/scan', { method: 'POST' }).then((r) => r.json()).catch(() => ({ findings: [] })),
      ]);

      if (resStatus) setStatus(resStatus);
      if (Array.isArray(resHistory)) setHistory(resHistory);
      if (resDiff) setDiffData(resDiff);
      if (resConfig) setConfigData(resConfig);
      if (resScan && resScan.findings) setFindings(resScan.findings);
    } catch (err) {
      console.error('Failed to fetch studio data:', err);
    } finally {
      setLoading(false);
    }
  };

  const triggerScan = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/scan', { method: 'POST' }).then((r) => r.json());
      if (res && res.findings) {
        setFindings(res.findings);
      }
    } catch (err) {
      console.error('Scan failed:', err);
    } finally {
      setLoading(false);
    }
  };

  const runHook = async (hookName: string) => {
    setRunningHook(true);
    try {
      await fetch('/api/run', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ hook_name: hookName }),
      });
      await fetchData();
    } catch (err) {
      console.error('Failed to run hook:', err);
    } finally {
      setRunningHook(false);
    }
  };

  const stageFiles = async () => {
    setStaging(true);
    try {
      await fetch('/api/stage', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({}),
      });
      await fetchData();
    } catch (err) {
      console.error('Failed to stage files:', err);
    } finally {
      setStaging(false);
    }
  };

  const clearCache = async () => {
    setClearingCache(true);
    try {
      await fetch('/api/cache/clean', { method: 'POST' });
      await fetchData();
    } catch (err) {
      console.error('Failed to clear cache:', err);
    } finally {
      setClearingCache(false);
    }
  };

  useEffect(() => {
    fetchData();
  }, []);

  return (
    <div className="min-h-screen bg-[#090d16] text-slate-100 flex flex-col">
      <Header
        status={status}
        activeTab={activeTab}
        setActiveTab={setActiveTab}
        loading={loading}
        onRefresh={fetchData}
        onRunHook={runHook}
        runningHook={runningHook}
      />

      <main className="flex-1 max-w-7xl w-full mx-auto p-6 space-y-6">
        <MetricCards status={status} history={history} />

        {activeTab === 'overview' && (
          <div className="space-y-6">
            <WaterfallChart history={history} />
          </div>
        )}

        {activeTab === 'secrets' && (
          <SecretShieldView
            findings={findings}
            loading={loading}
            onScan={triggerScan}
            entropyThreshold={configData?.config?.settings?.entropy_threshold || 3.8}
          />
        )}

        {activeTab === 'diff' && (
          <StagedDiffView diffData={diffData} onStageFiles={stageFiles} staging={staging} />
        )}

        {activeTab === 'config' && (
          <ConfigEditor configData={configData} onClearCache={clearCache} clearingCache={clearingCache} />
        )}
      </main>

      <footer className="border-t border-[#1f293d] py-4 text-center text-xs text-slate-500 font-mono">
        HookCraft &bull; Ultra-Fast Zero-Dependency Git Hook Orchestrator &bull; MIT License
      </footer>
    </div>
  );
};

export default App;
