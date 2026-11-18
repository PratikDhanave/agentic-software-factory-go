'use client';

import React, { useEffect, useMemo, useState } from 'react';
import { ArrowRight, Command, Search, X } from 'lucide-react';

export interface PaletteAction {
  id: string;
  group: string;
  label: string;
  hint?: string;
  onSelect: () => void;
}

interface Props {
  open: boolean;
  onClose: () => void;
  actions: PaletteAction[];
}

export default function CommandPalette({ open, onClose, actions }: Props) {
  const [query, setQuery] = useState('');
  const [activeIndex, setActiveIndex] = useState(0);

  useEffect(() => {
    if (open) {
      setQuery('');
      setActiveIndex(0);
    }
  }, [open]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return actions;
    return actions.filter(
      (a) =>
        a.label.toLowerCase().includes(q) ||
        a.group.toLowerCase().includes(q) ||
        (a.hint && a.hint.toLowerCase().includes(q))
    );
  }, [actions, query]);

  useEffect(() => {
    if (!open) return;
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault();
        onClose();
      } else if (e.key === 'ArrowDown') {
        e.preventDefault();
        setActiveIndex((i) => (filtered.length ? (i + 1) % filtered.length : 0));
      } else if (e.key === 'ArrowUp') {
        e.preventDefault();
        setActiveIndex((i) => (filtered.length ? (i - 1 + filtered.length) % filtered.length : 0));
      } else if (e.key === 'Enter' && filtered[activeIndex]) {
        e.preventDefault();
        filtered[activeIndex].onSelect();
        onClose();
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [open, filtered, activeIndex, onClose]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-50 bg-slate-950/40 backdrop-blur-[2px] flex items-start justify-center pt-[12vh] px-4"
      onClick={onClose}
    >
      <div
        className="w-full max-w-xl ide-surface border ide-border rounded-xl shadow-2xl overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center gap-2.5 px-4 py-3 border-b ide-border">
          <Search className="w-4 h-4 ide-ink-muted shrink-0" />
          <input
            autoFocus
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setActiveIndex(0);
            }}
            placeholder="Jump to view, launch a blueprint build, or switch mode..."
            className="flex-1 bg-transparent text-sm ide-ink placeholder:ide-ink-muted focus:outline-none"
          />
          <button
            type="button"
            onClick={onClose}
            className="p-1 rounded hover:bg-black/5 dark:hover:bg-white/5 ide-ink-muted"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="max-h-80 overflow-y-auto p-2 space-y-0.5">
          {filtered.length === 0 ? (
            <div className="py-8 text-center text-xs ide-ink-muted">
              No matching commands for &ldquo;{query}&rdquo;
            </div>
          ) : (
            filtered.map((item, idx) => {
              const active = idx === activeIndex;
              return (
                <button
                  key={item.id}
                  type="button"
                  onMouseEnter={() => setActiveIndex(idx)}
                  onClick={() => {
                    item.onSelect();
                    onClose();
                  }}
                  className={`w-full flex items-center justify-between gap-3 px-3 py-2 rounded-lg text-left text-xs transition ${
                    active
                      ? 'bg-blue-600 text-white'
                      : 'ide-ink hover:bg-black/5 dark:hover:bg-white/5'
                  }`}
                >
                  <div className="flex items-center gap-2.5 min-w-0">
                    <span
                      className={`text-[10px] uppercase tracking-wider font-semibold px-1.5 py-0.5 rounded ${
                        active ? 'bg-white/20 text-white' : 'ide-subtle ide-ink-muted'
                      }`}
                    >
                      {item.group}
                    </span>
                    <span className="font-medium truncate">{item.label}</span>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    {item.hint && (
                      <span className={`font-mono text-[11px] ${active ? 'text-blue-100' : 'ide-ink-muted'}`}>
                        {item.hint}
                      </span>
                    )}
                    <ArrowRight className={`w-3.5 h-3.5 ${active ? 'text-white' : 'ide-ink-muted'}`} />
                  </div>
                </button>
              );
            })
          )}
        </div>

        <div className="px-4 py-2 border-t ide-border ide-subtle flex items-center justify-between text-[11px] ide-ink-muted">
          <span className="inline-flex items-center gap-1">
            <Command className="w-3 h-3" /> Use ↑↓ to navigate, Enter to run
          </span>
          <span>ESC to close</span>
        </div>
      </div>
    </div>
  );
}
