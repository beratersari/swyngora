/** Close of the clicked candle, clamped to now while that candle is still forming.
 * A forming candle stays on the current minute so the request does not change every render.
 * A closed candle stays on its close. */
export function situationCandleCloseMs(
  openSec: number,
  barSec: number,
  nowMs: number,
): number | null {
  if (!Number.isFinite(openSec) || !Number.isFinite(barSec) || barSec <= 0 || !Number.isFinite(nowMs)) {
    return null;
  }
  const closeMs = (openSec + barSec) * 1000;
  if (nowMs < closeMs) {
    return Math.floor(nowMs / 60_000) * 60_000;
  }
  return closeMs;
}
