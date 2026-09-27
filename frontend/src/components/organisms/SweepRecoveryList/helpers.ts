export function formatRecovery(seconds: number | undefined): string {
  const n = seconds ?? 0;
  if (n < 60) return `${n}s`;
  if (n < 3600) return `${Math.round(n / 60)}m`;
  const hours = Math.floor(n / 3600);
  const mins = Math.round((n % 3600) / 60);
  return `${hours}h ${mins}m`;
}
