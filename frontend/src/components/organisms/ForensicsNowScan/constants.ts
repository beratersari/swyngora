export const SCAN_FIELD_ORDER = ['price', 'volume', 'takers', 'oi', 'book'] as const;

export type ScanFieldName = (typeof SCAN_FIELD_ORDER)[number];
