/** One stored-history reading. The English `summary` is for the assistant, not the page. */
export type UnusualReadingView = {
  metric?: string;
  band?: string;
  reason?: string;
  current?: number;
  median?: number;
  percentile?: number;
  samples?: number;
  minSamples?: number;
  spanHours?: number;
  unit?: string;
  label?: string;
  summary?: string;
};

export type UnusualVenueView = {
  exchange?: string;
  readings?: UnusualReadingView[];
};

export type UnusualReportView = {
  symbol?: string;
  exchange?: string;
  venues?: UnusualVenueView[];
  note?: string;
};

export type UnusualContextProps = {
  report?: UnusualReportView | null;
  isLoading?: boolean;
  error?: string | null;
};

export type UnusualTranslate = (key: string, options?: Record<string, string | number>) => string;
