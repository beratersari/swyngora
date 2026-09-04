import { env } from '@/config/env';
import type { ImportJob } from '@/libs/api';
import { getOrCreateClientId } from '@/libs/utils';

export type ImportProfileConflictView = {
  name?: string;
  incoming?: { index?: number; isDefault?: boolean; weights?: Record<string, number> }[];
  candidates?: { id?: string; name?: string; isDefault?: boolean; updatedAt?: string }[];
};

export type CustomSignalImportPreview = {
  addNames: string[];
  updateNames: string[];
  conflicts: ImportProfileConflictView[];
  willAdd: number;
  willUpdate: number;
  willConflict: number;
};

export type CustomSignalConflictPick = {
  incomingIndex?: number;
  profileId?: string;
};

export type CustomSignalResolutionPayload = {
  name: string;
  incomingIndex?: number;
  profileId?: string;
};

/** Named custom-signal mixes from an import preview (new vs overwrite vs conflict). */
export function customSignalImportPreview(job?: ImportJob | null): CustomSignalImportPreview {
  const sec = job?.sections?.customSignals;
  return {
    addNames: [...(sec?.addNames ?? [])],
    updateNames: [...(sec?.updateNames ?? [])],
    conflicts: [...(sec?.conflicts ?? [])],
    willAdd: sec?.willAdd ?? 0,
    willUpdate: sec?.willUpdate ?? 0,
    willConflict: sec?.willConflict ?? 0,
  };
}

/** Builds confirm payload; names without a required pick are omitted (skipped). */
export function customSignalResolutionsFromPicks(
  conflicts: ImportProfileConflictView[],
  picks: Record<string, CustomSignalConflictPick>,
): CustomSignalResolutionPayload[] {
  const out: CustomSignalResolutionPayload[] = [];
  for (const conflict of conflicts) {
    const name = conflict.name?.trim();
    if (!name) continue;
    const pick = picks[name];
    if (!pick) continue;
    const needIncoming = (conflict.incoming?.length ?? 0) > 1;
    const needTarget = (conflict.candidates?.length ?? 0) > 1;
    if (needIncoming && pick.incomingIndex === undefined) continue;
    if (needTarget && !pick.profileId) continue;
    if (!needIncoming && !needTarget) continue;
    const row: CustomSignalResolutionPayload = { name };
    if (needIncoming) row.incomingIndex = pick.incomingIndex;
    if (needTarget && pick.profileId) row.profileId = pick.profileId;
    out.push(row);
  }
  return out;
}

export function exportDownloadHref(downloadUrl?: string | null): string | null {
  if (!downloadUrl) return null;
  if (/^https?:\/\//i.test(downloadUrl)) return downloadUrl;
  const base = env.apiBaseUrl.replace(/\/+$/, '');
  const path = downloadUrl.startsWith('/') ? downloadUrl : `/${downloadUrl}`;
  return `${base}${path}`;
}

export function currentClientId(): string {
  return getOrCreateClientId();
}

/** Trade keys may become the browser session. Read keys must not — they 403 every mutation. */
export function createdKeySessionToken(
  secret?: string | null,
  permission?: string | null,
): string | null {
  const token = (secret ?? '').trim();
  if (!token) return null;
  if ((permission ?? '').trim().toLowerCase() !== 'trade') return null;
  return token;
}
