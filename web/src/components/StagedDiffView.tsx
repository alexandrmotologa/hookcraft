import React, { useState } from 'react';
import { FileCode, Layers, CheckSquare } from 'lucide-react';
import { DiffResponse, FileHunk } from '../types';

interface StagedDiffViewProps {
  diffData: DiffResponse | null;
  onStageFiles?: () => Promise<void>;
  staging?: boolean;
}

export const StagedDiffView: React.FC<StagedDiffViewProps> = ({ diffData, onStageFiles, staging }) => {
  const [selectedFile, setSelectedFile] = useState<string | null>(null);

  if (!diffData || !diffData.hunks || diffData.hunks.length === 0) {
    return (
      <div className="p-12 rounded-xl bg-[#111827] border border-[#1f293d] text-center">
        <div className="w-12 h-12 rounded-full bg-slate-800/60 border border-slate-700/60 flex items-center justify-center text-slate-400 mx-auto mb-3">
          <Layers className="w-6 h-6" />
        </div>
        <h3 className="font-semibold text-white">No Changes Staged</h3>
        <p className="text-xs text-slate-400 mt-1 max-w-sm mx-auto">
          Stage modifications using <code className="text-cyan-400 font-mono">git add &lt;file&gt;</code> to inspect diff hunks before committing.
        </p>
      </div>
    );
  }

  const hunks = diffData.hunks || [];
  const activeHunk = selectedFile
    ? hunks.find((h) => h.FilePath === selectedFile) || hunks[0]
    : hunks[0];

  const addedLines = activeHunk?.AddedLines || [];

  return (
    <div className="space-y-3">
      {onStageFiles && (
        <div className="flex justify-end">
          <button
            onClick={onStageFiles}
            disabled={staging}
            className="flex items-center space-x-1.5 px-3 py-1.5 rounded-lg bg-cyan-600 hover:bg-cyan-500 text-white text-xs font-semibold shadow-sm shadow-cyan-600/30 transition-all disabled:opacity-50"
          >
            <CheckSquare className="w-3.5 h-3.5" />
            <span>{staging ? 'Staging...' : 'Stage Modified Changes'}</span>
          </button>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-4">
        {/* File List */}
        <div className="lg:col-span-1 rounded-xl bg-[#111827] border border-[#1f293d] p-3 space-y-2">
        <div className="px-2 py-1 text-xs font-semibold text-slate-400 uppercase tracking-wider">
          Staged Files ({hunks.length})
        </div>
        <div className="space-y-1">
          {hunks.map((hunk: FileHunk) => {
            const isSelected = activeHunk.FilePath === hunk.FilePath;
            return (
              <button
                key={hunk.FilePath}
                onClick={() => setSelectedFile(hunk.FilePath)}
                className={`w-full text-left px-3 py-2 rounded-lg text-xs font-mono transition-all flex items-center justify-between ${
                  isSelected
                    ? 'bg-cyan-500/15 text-cyan-300 border border-cyan-500/40 font-semibold'
                    : 'text-slate-300 hover:bg-slate-800/50 border border-transparent'
                }`}
              >
                <div className="flex items-center space-x-2 truncate">
                  <FileCode className="w-3.5 h-3.5 shrink-0 text-slate-400" />
                  <span className="truncate">{hunk.FilePath}</span>
                </div>
                {hunk.IsNew && (
                  <span className="px-1.5 py-0.2 text-[9px] rounded bg-emerald-950 text-emerald-400 border border-emerald-800/40">
                    new
                  </span>
                )}
              </button>
            );
          })}
        </div>
      </div>

      {/* Diff Inspector */}
      <div className="lg:col-span-3 rounded-xl bg-[#111827] border border-[#1f293d] overflow-hidden">
        <div className="p-4 border-b border-[#1f293d] flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <span className="text-xs font-mono text-cyan-400">{activeHunk.FilePath}</span>
            {activeHunk.IsNew && (
              <span className="px-2 py-0.5 text-[10px] rounded bg-emerald-950 text-emerald-400 border border-emerald-800/40">
                Created
              </span>
            )}
          </div>
          <span className="text-xs font-mono text-slate-400">
            {addedLines.length} added lines
          </span>
        </div>

        <div className="p-4 font-mono text-xs overflow-x-auto bg-[#090d16] space-y-0.5">
          {addedLines.length === 0 ? (
            <div className="py-6 text-center text-slate-500 italic">
              No added lines in hunk.
            </div>
          ) : (
            addedLines.map((line, idx) => (
              <div
                key={idx}
                className="flex items-start hover:bg-slate-800/40 py-0.5 px-2 rounded group"
              >
                <span className="w-12 text-slate-500 select-none shrink-0 text-right pr-4 group-hover:text-slate-400">
                  {line.LineNumber}
                </span>
                <span className="text-emerald-400 pr-2 select-none font-bold">+</span>
                <span className="text-emerald-200/90 whitespace-pre break-all">{line.Content}</span>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
    </div>
  );
};
