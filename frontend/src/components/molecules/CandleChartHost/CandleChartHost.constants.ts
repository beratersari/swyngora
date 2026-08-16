export const DEFAULT_HEIGHT = 280;

/** Fire history load when the left of the visible logical range is within this many bars of index 0. */
export const HISTORY_LOAD_THRESHOLD = 20;

/** First paint shows this many latest bars (right-aligned), not the full series. */
export const INITIAL_VISIBLE_BARS = 80;

/** Empty slots to the right of the last bar so the newest candle is not flush. */
export const INITIAL_RIGHT_PADDING = 6;
