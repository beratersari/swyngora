import { CUSTOM_SIGNAL_FACTORS, type CustomSignalFactorId } from './constants';
import type { CustomSignalItem } from '@/libs/api';
import type { CustomMixPreviewFactor, CustomMixPreviewRow, CustomWeightDraftRow } from './SignalsCustomPanel.types';

export function defaultCustomWeightDraft(): CustomWeightDraftRow[] {
  return CUSTOM_SIGNAL_FACTORS.map((f) => ({
    id: f.id,
    enabled: f.defaultPct > 0,
    pct: f.defaultPct,
  }));
}

export function draftFromWeights(weights?: Record<string, number>): CustomWeightDraftRow[] {
  return CUSTOM_SIGNAL_FACTORS.map((f) => {
    const pct = Number(weights?.[f.id] ?? 0);
    return { id: f.id, enabled: pct > 0, pct: Number.isFinite(pct) ? pct : 0 };
  });
}

export function weightsFromDraft(draft: CustomWeightDraftRow[]): Record<string, number> {
  const out: Record<string, number> = {};
  for (const row of draft) {
    out[row.id] = row.enabled ? row.pct : 0;
  }
  return out;
}

export function customWeightTotal(draft: CustomWeightDraftRow[]): number {
  return draft.reduce((sum, row) => sum + (row.enabled && Number.isFinite(row.pct) ? row.pct : 0), 0);
}

export function customWeightsValid(draft: CustomWeightDraftRow[]): boolean {
  return Math.abs(customWeightTotal(draft) - 100) <= 0.05 && draft.some((r) => r.enabled && r.pct > 0);
}

export function draftNeedsHunt(draft: CustomWeightDraftRow[]): boolean {
  const row = draft.find((d) => d.id === 'hunt_up');
  return Boolean(row?.enabled && row.pct > 0);
}

function round1(v: number): number {
  return Math.round(v * 10) / 10;
}

function clampScore(v: number): number {
  if (!Number.isFinite(v)) return 0;
  return round1(Math.max(0, Math.min(100, v)));
}

export function customSignalGrade(score: number): string {
  if (score >= 78) return 'A';
  if (score >= 62) return 'B';
  return 'C';
}

function pctOf(draft: CustomWeightDraftRow[], id: CustomSignalFactorId): number {
  const row = draft.find((d) => d.id === id);
  if (!row?.enabled) return 0;
  return Number.isFinite(row.pct) ? row.pct : 0;
}

function assemble(
  item: CustomSignalItem | undefined,
  pct: (id: CustomSignalFactorId) => number,
): { score: number; effects: Map<string, number>; statuses: Map<string, CustomMixPreviewFactor['status']> } {
  let raw = 50;
  const effects = new Map<string, number>();
  const statuses = new Map<string, CustomMixPreviewFactor['status']>();
  for (const factor of CUSTOM_SIGNAL_FACTORS) {
    const src = item?.custom?.factors?.find((f) => f.id === factor.id);
    const share = pct(factor.id);
    if (share <= 0) {
      statuses.set(factor.id, 'disabled');
      effects.set(factor.id, 0);
      continue;
    }
    if (src?.status === 'missing') {
      statuses.set(factor.id, 'missing');
      effects.set(factor.id, 0);
      continue;
    }
    const score = Number.isFinite(src?.score) ? (src?.score ?? 0) : 0;
    const effect = (score - 50) * (share / 100);
    raw += effect;
    statuses.set(factor.id, 'used');
    effects.set(factor.id, round1(effect));
  }
  return { score: clampScore(raw), effects, statuses };
}

export function largestCustomFactorChange(factors: CustomMixPreviewFactor[]): CustomMixPreviewFactor | null {
  let best: CustomMixPreviewFactor | null = null;
  for (const factor of factors) {
    if (Math.abs(factor.deltaEffect) < 0.05) continue;
    if (!best || Math.abs(factor.deltaEffect) > Math.abs(best.deltaEffect)) {
      best = factor;
    }
  }
  return best;
}

export function previewCustomMix(items: CustomSignalItem[], draft: CustomWeightDraftRow[]): CustomMixPreviewRow[] {
  const def = defaultCustomWeightDraft();
  const defaultPct = (id: CustomSignalFactorId) => pctOf(def, id);
  const appliedPct = (id: CustomSignalFactorId) => pctOf(draft, id);
  return items.map((item) => {
    const assembledDef = assemble(item, defaultPct);
    const assembledApp = assemble(item, appliedPct);
    const factors: CustomMixPreviewFactor[] = CUSTOM_SIGNAL_FACTORS.map((factor) => {
      const src = item.custom?.factors?.find((f) => f.id === factor.id);
      return {
        id: factor.id,
        score: src?.score ?? 0,
        status: assembledApp.statuses.get(factor.id) ?? 'missing',
        defaultPct: defaultPct(factor.id),
        appliedPct: appliedPct(factor.id),
        defaultEffect: assembledDef.effects.get(factor.id) ?? 0,
        appliedEffect: assembledApp.effects.get(factor.id) ?? 0,
        deltaEffect: round1((assembledApp.effects.get(factor.id) ?? 0) - (assembledDef.effects.get(factor.id) ?? 0)),
        detail: src?.detail,
      };
    });
    const hunt = factors.find((f) => f.id === 'hunt_up');
    return {
      exchange: item.exchange ?? '',
      symbol: item.symbol ?? '',
      interval: item.interval,
      price: item.price,
      defaultScore: assembledDef.score,
      appliedScore: assembledApp.score,
      delta: round1(assembledApp.score - assembledDef.score),
      grade: customSignalGrade(assembledApp.score),
      factors,
      largestChange: largestCustomFactorChange(factors),
      huntMissing: Boolean(hunt && hunt.appliedPct > 0 && hunt.status === 'missing'),
    };
  }).sort((a, b) => b.appliedScore - a.appliedScore || a.symbol.localeCompare(b.symbol));
}

export function scanWeightParams(draft: CustomWeightDraftRow[]): Record<string, number> {
  const out: Record<string, number> = {};
  for (const factor of CUSTOM_SIGNAL_FACTORS) {
    const row = draft.find((d) => d.id === factor.id);
    out[factor.query] = row?.enabled ? row.pct : 0;
  }
  return out;
}
