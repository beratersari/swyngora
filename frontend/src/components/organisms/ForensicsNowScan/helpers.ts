export function setupRadarAlertHref(symbol: string, exchange?: string): string {
  const q = new URLSearchParams({ kind: 'setup_radar', symbol: symbol || 'ALL' });
  if (exchange) q.set('exchange', exchange);
  return `/alerts?${q.toString()}`;
}
