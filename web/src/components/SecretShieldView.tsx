import React, { useState } from 'react';
import { Shield, ShieldCheck, RefreshCw, Key, AlertTriangle, FileCode } from 'lucide-react';
import { SecretFinding } from '../types';

interface SecretShieldViewProps {
  findings: SecretFinding[];
  loading: boolean;
  onScan: () => void;
  entropyThreshold: number;
}

export const SecretShieldView: React.FC<SecretShieldViewProps> = ({
  findings,
  loading,
  onScan,
  entropyThreshold,
}) => {
  const [filter, setFilter] = useState('');

  const filteredFindings = findings.filter(
    (f) =>
      f.file_path.toLowerCase().includes(filter.toLowerCase()) ||
      f.rule_name.toLowerCase().includes(filter.toLowerCase()) ||
      f.description.toLowerCase().includes(filter.toLowerCase())
  );

  return (
    <div className="space-y-4">
      {/* Header bar */}
      <div className="p-4 rounded-xl bg-[#111827] border border-[#1f293d] flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <div className="flex items-center space-x-2">
            <Shield className="w-5 h-5 text-cyan-400" />
            <h3 className="font-bold text-white tracking-tight">Staged Secret Leak Shield</h3>
            <span className="px-2 py-0.5 text-xs font-mono rounded-full bg-cyan-950 text-cyan-400 border border-cyan-800/60">
              Shannon Entropy &ge; {entropyThreshold}
            </span>
          </div>
          <p className="text-xs text-slate-400 mt-1">
            Detects AWS, OpenAI, GitHub tokens, private keys, and high-entropy credentials before commit creation.
          </p>
        </div>

        <div className="flex items-center space-x-3">
          <input
            type="text"
            placeholder="Filter findings..."
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="px-3 py-1.5 rounded-lg bg-[#090d16] border border-[#1f293d] text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-cyan-500"
          />
          <button
            onClick={onScan}
            disabled={loading}
            className="flex items-center space-x-2 px-3 py-1.5 rounded-lg bg-cyan-600 hover:bg-cyan-500 text-white text-xs font-medium transition-all shadow-sm shadow-cyan-600/20 disabled:opacity-50"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? 'animate-spin' : ''}`} />
            <span>Scan Staged</span>
          </button>
        </div>
      </div>

      {/* Findings List */}
      {findings.length === 0 ? (
        <div className="p-12 rounded-xl bg-[#111827] border border-[#1f293d] text-center">
          <div className="w-12 h-12 rounded-full bg-emerald-950/60 border border-emerald-800/60 flex items-center justify-center text-emerald-400 mx-auto mb-3">
            <ShieldCheck className="w-6 h-6" />
          </div>
          <h3 className="font-semibold text-white">Staged Diffs Clean</h3>
          <p className="text-xs text-slate-400 mt-1 max-w-md mx-auto">
            No API keys, credentials, or high-entropy tokens were detected in your staged changes.
          </p>
        </div>
      ) : (
        <div className="space-y-3">
          <div className="p-3 rounded-lg bg-rose-950/40 border border-rose-900/60 flex items-center space-x-3 text-rose-300 text-xs">
            <AlertTriangle className="w-4 h-4 shrink-0 text-rose-400" />
            <span>
              <strong>{findings.length} secret leak(s) detected.</strong> Commits will be blocked until these tokens are removed or bypassed with inline comments.
            </span>
          </div>

          <div className="grid grid-cols-1 gap-3">
            {filteredFindings.map((finding, idx) => (
              <div
                key={idx}
                className="p-4 rounded-xl bg-[#111827] border border-rose-900/40 hover:border-rose-700/60 transition-all space-y-3"
              >
                <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
                  <div className="flex items-center space-x-2">
                    <span className="p-1 rounded bg-rose-950 text-rose-400 border border-rose-800/60">
                      <Key className="w-3.5 h-3.5" />
                    </span>
                    <span className="font-bold text-sm text-white">{finding.rule_name}</span>
                    <span className="text-xs text-slate-400 font-mono">({finding.rule_id})</span>
                  </div>

                  <div className="flex items-center space-x-2 text-xs font-mono">
                    <span className="px-2 py-0.5 rounded bg-[#090d16] text-cyan-400 border border-[#1f293d] flex items-center gap-1">
                      <FileCode className="w-3 h-3" />
                      {finding.file_path}:{finding.line_number}
                    </span>
                    {finding.entropy && (
                      <span className="px-2 py-0.5 rounded bg-[#090d16] text-purple-400 border border-[#1f293d]">
                        H={finding.entropy.toFixed(2)} b/B
                      </span>
                    )}
                  </div>
                </div>

                <p className="text-xs text-slate-400">{finding.description}</p>

                {/* Code snippet with redacted secret */}
                <div className="rounded-lg bg-[#090d16] border border-[#1f293d] p-3 font-mono text-xs overflow-x-auto space-y-1">
                  <div className="text-slate-400 flex items-center justify-between">
                    <span>Line {finding.line_number}:</span>
                    <span className="text-rose-400 font-bold tracking-wider">{finding.redacted_value}</span>
                  </div>
                  <pre className="text-slate-200 mt-1 whitespace-pre-wrap">{finding.raw_line}</pre>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};
