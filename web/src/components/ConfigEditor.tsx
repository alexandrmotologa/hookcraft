import React, { useState } from 'react';
import { Terminal, Copy, Check, Sliders, Shield, Zap } from 'lucide-react';
import { ConfigResponse } from '../types';

interface ConfigEditorProps {
  configData: ConfigResponse | null;
}

export const ConfigEditor: React.FC<ConfigEditorProps> = ({ configData }) => {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    if (configData?.raw_yaml) {
      navigator.clipboard.writeText(configData.raw_yaml);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const settings = configData?.config?.settings;

  return (
    <div className="space-y-4">
      {/* Settings Grid */}
      {settings && (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
          <div className="p-3.5 rounded-xl bg-[#111827] border border-[#1f293d]">
            <div className="flex items-center space-x-2 text-slate-400 text-xs font-medium">
              <Zap className="w-3.5 h-3.5 text-cyan-400" />
              <span>Concurrency</span>
            </div>
            <div className="mt-2 text-lg font-bold font-mono text-white">
              {settings.concurrency ? `Parallel (${settings.max_workers} CPU cores)` : 'Sequential'}
            </div>
          </div>

          <div className="p-3.5 rounded-xl bg-[#111827] border border-[#1f293d]">
            <div className="flex items-center space-x-2 text-slate-400 text-xs font-medium">
              <Sliders className="w-3.5 h-3.5 text-emerald-400" />
              <span>Auto-Stage Fixes</span>
            </div>
            <div className="mt-2 text-lg font-bold font-mono text-white">
              {settings.auto_stage_fixes ? 'Enabled' : 'Manual'}
            </div>
          </div>

          <div className="p-3.5 rounded-xl bg-[#111827] border border-[#1f293d]">
            <div className="flex items-center space-x-2 text-slate-400 text-xs font-medium">
              <Shield className="w-3.5 h-3.5 text-purple-400" />
              <span>Secret Shield</span>
            </div>
            <div className="mt-2 text-lg font-bold font-mono text-white">
              {settings.block_secret_leaks ? 'Blocking' : 'Warning'}
            </div>
          </div>

          <div className="p-3.5 rounded-xl bg-[#111827] border border-[#1f293d]">
            <div className="flex items-center space-x-2 text-slate-400 text-xs font-medium">
              <Terminal className="w-3.5 h-3.5 text-blue-400" />
              <span>Stash Guard</span>
            </div>
            <div className="mt-2 text-lg font-bold font-mono text-white">
              {settings.stash_unstaged ? 'Active' : 'Disabled'}
            </div>
          </div>
        </div>
      )}

      {/* Raw YAML Inspector */}
      <div className="rounded-xl bg-[#111827] border border-[#1f293d] overflow-hidden">
        <div className="p-4 border-b border-[#1f293d] flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <Terminal className="w-4 h-4 text-cyan-400" />
            <span className="font-bold text-sm text-white">.hookcraft.yaml</span>
            <span className="text-xs text-slate-400 font-mono">
              {configData?.path ? `(${configData.path})` : '(Auto-generated stack preset)'}
            </span>
          </div>

          <button
            onClick={handleCopy}
            className="flex items-center space-x-1.5 px-3 py-1.5 rounded-lg bg-[#0c1220] border border-[#1f293d] hover:border-slate-600 text-xs text-slate-300 transition-all"
          >
            {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
            <span>{copied ? 'Copied' : 'Copy YAML'}</span>
          </button>
        </div>

        <pre className="p-5 font-mono text-xs text-cyan-200/90 bg-[#090d16] overflow-x-auto whitespace-pre leading-relaxed">
          {configData?.raw_yaml || '# No configuration file loaded'}
        </pre>
      </div>
    </div>
  );
};
