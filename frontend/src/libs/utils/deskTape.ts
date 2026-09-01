export const DESK_TAPE_SOURCE_STORAGE_KEY = 'swyngora.deskTapeSource';

export const DESK_TAPE_SOURCES = ['binance', 'coinbase', 'bist', 'watchlist'] as const;

export type DeskTapeSource = (typeof DESK_TAPE_SOURCES)[number];

export const DESK_TAPE_VENUE_LIMIT = 36;

export function isDeskTapeSource(raw: string | null | undefined): raw is DeskTapeSource {
  return DESK_TAPE_SOURCES.includes((raw ?? '') as DeskTapeSource);
}

export function loadDeskTapeSource(): DeskTapeSource {
  try {
    const raw = localStorage.getItem(DESK_TAPE_SOURCE_STORAGE_KEY);
    if (isDeskTapeSource(raw)) return raw;
  } catch {
    /* ignore */
  }
  return 'binance';
}

export function saveDeskTapeSource(next: DeskTapeSource): void {
  try {
    localStorage.setItem(DESK_TAPE_SOURCE_STORAGE_KEY, next);
  } catch {
    /* ignore */
  }
}
