export const SIMILAR_FIELD_ORDER = ['price', 'volume', 'takers', 'oi', 'book', 'options'] as const;

export type SimilarFieldName = (typeof SIMILAR_FIELD_ORDER)[number];
