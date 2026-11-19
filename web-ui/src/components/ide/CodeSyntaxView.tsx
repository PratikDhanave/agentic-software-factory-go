'use client';

import React, { useMemo, useState } from 'react';
import { Check, Copy, Download, FileCode2, FileText, FolderOpen } from 'lucide-react';

export interface WorkspaceFileItem {
  path: string;
  lines?: number;
  content: string;
}

const GO_KEYWORDS = new Set([
  'package',
  'import',
  'func',
  'type',
  'struct',
  'interface',
  'var',
  'const',
  'return',
  'if',
  'else',
  'for',
  'range',
  'switch',
  'case',
  'default',
  'go',
  'defer',
  'select',
  'chan',
  'map',
  'break',
  'continue',
  'fallthrough',
]);

const GO_TYPES_AND_BUILTINS = new Set([
  'string',
  'int',
  'int64',
  'int32',
  'uint',
  'uint64',
  'float64',
  'float32',
  'bool',
  'byte',
  'rune',
  'error',
  'any',
  'comparable',
  'true',
  'false',
  'nil',
  'make',
  'len',
  'cap',
  'append',
  'delete',
  'close',
  'new',
  'panic',
  'recover',
]);

const TOKEN_REGEX =
  /(\/\/.*$|"(?:\\.|[^"\\])*"|`[^`]*`|\b(?:package|import|func|type|struct|interface|var|const|return|if|else|for|range|switch|case|default|go|defer|select|chan|map|break|continue|fallthrough|string|int|int64|int32|uint|uint64|float64|float32|bool|byte|rune|error|any|comparable|true|false|nil|make|len|cap|append|delete|close|new|panic|recover)\b|\b\d+(?:\.\d+)?\b)/gm;

function renderHighlightedLine(line: string): React.ReactNode {
  if (!line) return '\u00A0';
  const trimmed = line.trimStart();
  if (trimmed.startsWith('//') || trimmed.startsWith('#')) {
    return <span className="text-slate-400 dark:text-slate-500 italic">{line}</span>;
  }

  const parts: React.ReactNode[] = [];
  let lastIndex = 0;
  let match: RegExpExecArray | null;
  const regex = new RegExp(TOKEN_REGEX.source, 'g');

  while ((match = regex.exec(line)) !== null) {
    if (match.index > lastIndex) {
      parts.push(line.slice(lastIndex, match.index));
    }
    const token = match[0];
    let cls = 'ide-ink';
    if (token.startsWith('//')) {
      cls = 'text-slate-400 dark:text-slate-500 italic';
    } else if (token.startsWith('"') || token.startsWith('`')) {
      cls = 'text-emerald-700 dark:text-emerald-400';
    } else if (GO_KEYWORDS.has(token)) {
      cls = 'text-rose-600 dark:text-rose-400 font-medium';
    } else if (GO_TYPES_AND_BUILTINS.has(token)) {
      cls = 'text-blue-700 dark:text-sky-400';
    } else if (/^\d/.test(token)) {
      cls = 'text-amber-700 dark:text-amber-300';
    }
    parts.push(
      <span key={`${match.index}-${token}`} className={cls}>
        {token}
      </span>
    );
    lastIndex = regex.lastIndex;
  }

  if (lastIndex < line.length) {
    parts.push(line.slice(lastIndex));
  }

  return parts;
}

interface Props {
  files: WorkspaceFileItem[];
  workspaceDir?: string;
  promotedDir?: string;
}

export default function CodeSyntaxView({ files, workspaceDir, promotedDir }: Props) {
  const [selectedIdx, setSelectedIdx] = useState(0);
  const [copied, setCopied] = useState(false);

  const safeIdx = selectedIdx < files.length ? selectedIdx : 0;
  const activeFile = files[safeIdx] || null;

  const lines = useMemo(() => {
    if (!activeFile?.content) return [];
    return activeFile.content.replace(/\r\n/g, '\n').split('\n');
  }, [activeFile]);

  const handleCopy = async () => {
    if (!activeFile) return;
    await navigator.clipboard.writeText(activeFile.content);
    setCopied(true);
    setTimeout(() => setCopied(false), 1800);
  };

  const handleDownload = () => {
    if (!activeFile) return;
    const blob = new Blob([activeFile.content], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = activeFile.path.split('/').pop() || 'artifact.go';
    a.click();
    URL.revokeObjectURL(url);
  };

  if (files.length === 0) {
    return (
      <div className="ide-canvas border ide-border rounded-2xl p-12 text-center">
        <FileCode2 className="w-8 h-8 mx-auto ide-ink-muted mb-2.5" />
        <p className="text-sm font-medium ide-ink">No source files materialized yet</p>
        <p className="text-xs ide-ink-muted mt-1 max-w-md mx-auto">
          Once the run reaches Stage 04 (Write Code), generated Go implementation and unit test files appear here.
        </p>
      </div>
    );
  }

  return (
    <div className="ide-surface border ide-border rounded-2xl overflow-hidden grid grid-cols-1 md:grid-cols-12 min-h-[460px]">
      {/* Left File Explorer Tree */}
      <aside className="md:col-span-3 border-b md:border-b-0 md:border-r ide-border ide-canvas flex flex-col">
        <div className="px-4 py-2.5 border-b ide-border flex items-center justify-between">
          <span className="text-[11px] font-medium uppercase tracking-[0.06em] ide-ink-muted">
            Explorer ({files.length})
          </span>
        </div>

        <div className="p-2.5 space-y-1 flex-1 overflow-y-auto">
          <div className="flex items-center gap-1.5 px-2 py-1 text-[11px] font-medium ide-ink-secondary">
            <FolderOpen className="w-3.5 h-3.5 ide-ink-muted shrink-0" />
            <span className="truncate font-mono">
              {promotedDir ? promotedDir.split('/').slice(-2).join('/') : 'workspace/'}
            </span>
          </div>

          <div className="pl-2 space-y-0.5">
            {files.map((f, idx) => {
              const active = idx === safeIdx;
              const isTest = f.path.endsWith('_test.go');
              const lineCount = f.lines || f.content.split('\n').length;
              return (
                <button
                  key={f.path}
                  type="button"
                  onClick={() => setSelectedIdx(idx)}
                  className={`w-full flex items-center justify-between gap-2 px-3 py-1.5 rounded-full text-left text-xs transition ${
                    active
                      ? 'kora-nav-active'
                      : 'ide-ink-secondary hover:bg-black/[0.03] dark:hover:bg-white/[0.04]'
                  }`}
                >
                  <span className="flex items-center gap-2 min-w-0">
                    {isTest ? (
                      <FileText className="w-3.5 h-3.5 text-[#285936] shrink-0" />
                    ) : (
                      <FileCode2 className="w-3.5 h-3.5 ide-ink shrink-0" />
                    )}
                    <span className="font-mono truncate">{f.path}</span>
                  </span>
                  <span className="text-[10px] font-mono ide-ink-muted tabular-nums shrink-0">
                    {lineCount}L
                  </span>
                </button>
              );
            })}
          </div>
        </div>

        {workspaceDir && (
          <div className="px-3 py-2 border-t ide-border text-[10px] font-mono ide-ink-muted truncate" title={workspaceDir}>
            {workspaceDir.split('/').slice(-3).join('/')}
          </div>
        )}
      </aside>

      {/* Right Code Viewer with Tabs + Line Gutter */}
      <div className="md:col-span-9 flex flex-col min-w-0">
        {/* File Tab Bar + Actions */}
        <div className="ide-subtle border-b ide-border px-2 flex items-center justify-between gap-2 min-h-[38px]">
          <div className="flex items-center gap-1 overflow-x-auto py-1">
            {files.map((f, idx) => {
              const active = idx === safeIdx;
              return (
                <button
                  key={f.path}
                  type="button"
                  onClick={() => setSelectedIdx(idx)}
                  className={`px-3 py-1 rounded text-xs font-mono flex items-center gap-1.5 whitespace-nowrap transition ${
                    active
                      ? 'ide-surface ide-ink shadow-sm border ide-border font-medium'
                      : 'ide-ink-muted hover:ide-ink'
                  }`}
                >
                  <span>{f.path}</span>
                </button>
              );
            })}
          </div>

          <div className="flex items-center gap-1.5 shrink-0 pr-1">
            <button
              type="button"
              onClick={handleCopy}
              className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md ide-surface border ide-border text-xs ide-ink-secondary hover:ide-ink transition"
            >
              {copied ? <Check className="w-3.5 h-3.5 text-emerald-600" /> : <Copy className="w-3.5 h-3.5" />}
              {copied ? 'Copied' : 'Copy'}
            </button>
            <button
              type="button"
              onClick={handleDownload}
              className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md ide-surface border ide-border text-xs ide-ink-secondary hover:ide-ink transition"
              title="Download file"
            >
              <Download className="w-3.5 h-3.5" />
              Download
            </button>
          </div>
        </div>

        {/* Code Body */}
        <div
          className="flex-1 overflow-auto max-h-[540px] text-[12px] leading-[20px] font-mono"
          style={{ backgroundColor: 'var(--ide-code-bg)' }}
        >
          <table className="w-full border-collapse">
            <tbody>
              {lines.map((line, idx) => (
                <tr key={idx} className="hover:bg-blue-500/[0.04]">
                  <td
                    className="select-none text-right pr-3 pl-3 w-12 ide-ink-muted tabular-nums border-r ide-border align-top"
                    style={{ backgroundColor: 'var(--ide-gutter-bg)' }}
                  >
                    {idx + 1}
                  </td>
                  <td className="pl-4 pr-6 whitespace-pre ide-ink align-top">
                    {renderHighlightedLine(line)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
