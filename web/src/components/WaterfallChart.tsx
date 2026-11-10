import React, { useState } from 'react';
import { Clock, CheckCircle2, XCircle, SkipForward, HardDrive } from 'lucide-react';
import { RunSummary, StepResult } from '../types';

interface WaterfallChartProps {
  history: RunSummary[];
}

export const WaterfallChart: React.FC<WaterfallChartProps> = ({ history }) => {
  const [selectedRunIndex, setSelectedRunIndex] = useState(0);

  if (history.length === 0) {
    return (
      <div className="p-8 rounded-xl bg-[#111827] border border-[#1f293d] text-center">
        <Clock className="w-8 h-8 text-slate-500 mx-auto mb-2" />
        <h3 className="font-semibold text-slate-200">No Historical Runs Recorded</h3>
        <p className="text-xs text-slate-400 mt-1 max-w-sm mx-auto">
          Execute a git commit or run <code className="text-cyan-400">hookcraft run pre-commit</code> in your terminal to see execution timing waterfalls.
        </p>
      </div>
    );
  }

  const currentRun = history[selectedRunIndex] || history[0];
  const maxDuration = Math.max(...currentRun.steps.map((s) => s.duration_ms), 1);

  return (
    <div className="rounded-xl bg-[#111827] border border-[#1f293d] overflow-hidden">
      {/* Header & Run Selector */}
      <div className="p-5 border-b border-[#1f293d] flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <div className="flex items-center space-x-2">
            <h3 className="font-bold text-white tracking-tight">Hook Execution Waterfall</h3>
            <span className={`px-2 py-0.5 text-xs font-mono rounded-full border ${
              currentRun.success
                ? 'bg-emerald-950 text-emerald-400 border-emerald-800/60'
                : 'bg-rose-950 text-rose-400 border-rose-800/60'
            }`}>
              {currentRun.success ? 'PASSED' : 'FAILED'}
            </span>
          </div>
          <p className="text-xs text-slate-400 mt-0.5">
            Hook: <span className="font-mono text-cyan-400">{currentRun.hook_name}</span> | ID: <span className="font-mono text-slate-400">{currentRun.id}</span>
          </p>
        </div>

        {/* Run Selector Pills */}
        <div className="flex items-center space-x-2 overflow-x-auto py-1">
          <span className="text-xs text-slate-400 shrink-0 font-medium">Runs:</span>
          {history.slice(0, 5).map((run, idx) => (
            <button
              key={run.id}
              onClick={() => setSelectedRunIndex(idx)}
              className={`px-2.5 py-1 text-xs font-mono rounded-lg transition-all shrink-0 flex items-center space-x-1.5 border ${
                selectedRunIndex === idx
                  ? 'bg-cyan-500/20 text-cyan-300 border-cyan-500/50 shadow-sm'
                  : 'bg-[#0c1220] text-slate-400 border-[#1f293d] hover:text-slate-200'
              }`}
            >
              <span className={`w-1.5 h-1.5 rounded-full ${run.success ? 'bg-emerald-400' : 'bg-rose-400'}`} />
              <span>{run.duration_ms.toFixed(1)}ms</span>
            </button>
          ))}
        </div>
      </div>

      {/* Waterfall Visualization */}
      <div className="p-5 space-y-4">
        {currentRun.steps.map((step: StepResult, index: number) => {
          const percentage = Math.max(Math.min((step.duration_ms / maxDuration) * 100, 100), 4);

          return (
            <div key={index} className="space-y-1.5">
              <div className="flex items-center justify-between text-xs">
                <div className="flex items-center space-x-2">
                  {step.skipped ? (
                    <SkipForward className="w-3.5 h-3.5 text-amber-400 shrink-0" />
                  ) : step.success ? (
                    <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400 shrink-0" />
                  ) : (
                    <XCircle className="w-3.5 h-3.5 text-rose-400 shrink-0" />
                  )}
                  <span className="font-medium text-slate-200">{step.step_name}</span>
                  {step.skipped && (
                    <span className="text-[11px] text-amber-400 font-mono italic">({step.skip_reason})</span>
                  )}
                  {step.modified_files && step.modified_files.length > 0 && (
                    <span className="px-1.5 py-0.2 text-[10px] rounded bg-cyan-950 text-cyan-300 border border-cyan-800/40">
                      auto-staged {step.modified_files.length} file(s)
                    </span>
                  )}
                </div>
                <div className="flex items-center space-x-2 font-mono">
                  <span className="text-slate-400 text-[11px]">{step.duration_ms.toFixed(1)} ms</span>
                </div>
              </div>

              {/* Progress bar */}
              <div className="h-2.5 rounded-full bg-[#0c1220] border border-[#1f293d]/80 overflow-hidden flex items-center">
                <div
                  className={`h-full rounded-full transition-all duration-500 ${
                    step.skipped
                      ? 'bg-amber-500/40'
                      : step.success
                      ? 'bg-gradient-to-r from-cyan-500 to-emerald-400'
                      : 'bg-rose-500'
                  }`}
                  style={{ width: `${percentage}%` }}
                />
              </div>

              {/* Output detail if failed */}
              {!step.success && step.output && (
                <pre className="mt-2 p-2.5 rounded-lg bg-[#090d16] border border-rose-900/50 text-rose-300 text-xs font-mono whitespace-pre-wrap overflow-x-auto">
                  {step.output}
                </pre>
              )}
            </div>
          );
        })}
      </div>

      {/* Footer stats */}
      <div className="px-5 py-3 bg-[#0c1220] border-t border-[#1f293d] flex flex-wrap items-center justify-between text-xs text-slate-400 gap-3">
        <div className="flex items-center space-x-4">
          <span className="flex items-center gap-1.5">
            <Clock className="w-3.5 h-3.5 text-cyan-400" />
            Total Duration: <strong className="text-white font-mono">{currentRun.duration_ms.toFixed(1)} ms</strong>
          </span>
          {currentRun.total_memory_mb > 0 && (
            <span className="flex items-center gap-1.5">
              <HardDrive className="w-3.5 h-3.5 text-purple-400" />
              Memory Footprint: <strong className="text-white font-mono">{currentRun.total_memory_mb.toFixed(2)} MB</strong>
            </span>
          )}
        </div>
        <div className="text-[11px] text-slate-400 font-mono">
          {new Date(currentRun.timestamp).toLocaleTimeString()}
        </div>
      </div>
    </div>
  );
};
