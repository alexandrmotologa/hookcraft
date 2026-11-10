import React from 'react';
import { Shield, GitBranch, RefreshCw, Activity, FileCode, Terminal } from 'lucide-react';
import { StatusResponse } from '../types';

interface HeaderProps {
  status: StatusResponse | null;
  activeTab: 'overview' | 'secrets' | 'diff' | 'config';
  setActiveTab: (tab: 'overview' | 'secrets' | 'diff' | 'config') => void;
  loading: boolean;
  onRefresh: () => void;
}

export const Header: React.FC<HeaderProps> = ({
  status,
  activeTab,
  setActiveTab,
  loading,
  onRefresh,
}) => {
  return (
    <header className="border-b border-[#1f293d] bg-[#0c1220]/80 backdrop-blur-md sticky top-0 z-50 px-6 py-4">
      <div className="max-w-7xl mx-auto flex flex-col md:flex-row md:items-center md:justify-between gap-4">
        {/* Brand */}
        <div className="flex items-center space-x-3">
          <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-cyan-500 to-blue-600 flex items-center justify-center shadow-lg shadow-cyan-500/20 text-white font-bold">
            <Shield className="w-5 h-5 text-white" />
          </div>
          <div>
            <div className="flex items-center space-x-2">
              <span className="font-bold text-lg text-white tracking-tight">HookCraft</span>
              <span className="px-2 py-0.5 text-xs font-mono font-medium rounded-full bg-cyan-950 text-cyan-400 border border-cyan-800/60">
                v{status?.version || '1.0.0'}
              </span>
              <span className="px-2 py-0.5 text-xs font-mono font-medium rounded-full bg-emerald-950 text-emerald-400 border border-emerald-800/60 flex items-center gap-1">
                <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
                Sub-100ms
              </span>
            </div>
            <p className="text-xs text-slate-400">Zero-Dependency Git Hook Orchestrator & Secret Shield</p>
          </div>
        </div>

        {/* Navigation Tabs */}
        <nav className="flex items-center space-x-1 bg-[#111827] p-1 rounded-xl border border-[#1f293d]">
          <button
            onClick={() => setActiveTab('overview')}
            className={`flex items-center space-x-2 px-3 py-1.5 text-xs font-medium rounded-lg transition-all ${
              activeTab === 'overview'
                ? 'bg-cyan-500/10 text-cyan-400 border border-cyan-500/30 shadow-sm'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/50'
            }`}
          >
            <Activity className="w-3.5 h-3.5" />
            <span>Overview & Benchmark</span>
          </button>
          <button
            onClick={() => setActiveTab('secrets')}
            className={`flex items-center space-x-2 px-3 py-1.5 text-xs font-medium rounded-lg transition-all ${
              activeTab === 'secrets'
                ? 'bg-cyan-500/10 text-cyan-400 border border-cyan-500/30 shadow-sm'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/50'
            }`}
          >
            <Shield className="w-3.5 h-3.5" />
            <span>Secret Shield</span>
          </button>
          <button
            onClick={() => setActiveTab('diff')}
            className={`flex items-center space-x-2 px-3 py-1.5 text-xs font-medium rounded-lg transition-all ${
              activeTab === 'diff'
                ? 'bg-cyan-500/10 text-cyan-400 border border-cyan-500/30 shadow-sm'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/50'
            }`}
          >
            <FileCode className="w-3.5 h-3.5" />
            <span>Staged Diffs</span>
            {status && status.staged_count > 0 && (
              <span className="px-1.5 py-0.2 text-[10px] rounded-full bg-cyan-950 text-cyan-300 font-mono">
                {status.staged_count}
              </span>
            )}
          </button>
          <button
            onClick={() => setActiveTab('config')}
            className={`flex items-center space-x-2 px-3 py-1.5 text-xs font-medium rounded-lg transition-all ${
              activeTab === 'config'
                ? 'bg-cyan-500/10 text-cyan-400 border border-cyan-500/30 shadow-sm'
                : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/50'
            }`}
          >
            <Terminal className="w-3.5 h-3.5" />
            <span>Configuration</span>
          </button>
        </nav>

        {/* Right Info & Actions */}
        <div className="flex items-center space-x-3">
          {status && (
            <div className="hidden lg:flex items-center space-x-2 px-3 py-1.5 rounded-lg bg-[#111827] border border-[#1f293d] text-xs font-mono text-slate-300">
              <GitBranch className="w-3.5 h-3.5 text-cyan-400" />
              <span>{status.branch || 'detached'}</span>
            </div>
          )}

          <button
            onClick={onRefresh}
            disabled={loading}
            className="p-2 rounded-lg bg-[#111827] border border-[#1f293d] text-slate-400 hover:text-white hover:border-slate-700 transition-all disabled:opacity-50"
            title="Refresh repository state"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin text-cyan-400' : ''}`} />
          </button>
        </div>
      </div>
    </header>
  );
};
