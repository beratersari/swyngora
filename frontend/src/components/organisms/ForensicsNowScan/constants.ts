export const SCAN_FIELD_ORDER = ['price', 'volume', 'takers', 'oi', 'book', 'options'] as const;

export type ScanFieldName = (typeof SCAN_FIELD_ORDER)[number];
