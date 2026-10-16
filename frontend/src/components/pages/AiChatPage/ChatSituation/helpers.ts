import type { AiSituationEntry, AiSituationRead } from './ChatSituation.types';

type Translate = (key: string, options?: Record<string, unknown>) => string;

export function visibleSituationReads(reads: AiSituationRead[] | undefined): AiSituationRead[] {
  return (reads ?? []).filter((read) => Boolean(read?.symbol) && Boolean(read?.situation));
}

export function formatSituationTime(iso: string | undefined, locale: string): string {
  const trimmed = (iso ?? '').trim();
  if (!trimmed) return '';
  const date = new Date(trimmed);
  if (Number.isNaN(date.getTime())) return trimmed;
  return new Intl.DateTimeFormat(locale || 'en', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date);
}

/** Place and time of the open paper buy. Empty when the entry names neither. */
export function situationEntryCaption(entry: AiSituationEntry, t: Translate, locale: string): string {
  const place = [entry.book?.trim(), entry.exchange?.trim()].filter(Boolean).join(' · ');
  const time = formatSituationTime(entry.entryAt, locale);
  if (place && time) return t('situationEntryWhen', { place, time });
  if (place) return t('situationEntryWhere', { place });
  return '';
}
