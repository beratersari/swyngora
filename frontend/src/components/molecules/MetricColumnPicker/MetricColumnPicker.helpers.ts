/**
 * Move an item by index within a list (bounds-safe). Returns a new array.
 */
export function moveItemAtIndex<T>(items: readonly T[], from: number, to: number): T[] {
  if (from < 0 || from >= items.length) return [...items];
  if (to < 0 || to >= items.length) return [...items];
  if (from === to) return [...items];
  const next = [...items];
  const [item] = next.splice(from, 1);
  next.splice(to, 0, item);
  return next;
}

/** Move id one step toward the start (−1) or end (+1). */
export function moveIdByDelta<T extends string>(
  ids: readonly T[],
  id: T,
  delta: -1 | 1,
): T[] {
  const from = ids.indexOf(id);
  if (from < 0) return [...ids];
  return moveItemAtIndex(ids, from, from + delta);
}

/** Insert id at end if missing; remove if present. Keep at least one when removing. */
export function toggleIdInList<T extends string>(
  ids: readonly T[],
  id: T,
  options?: { minCount?: number },
): T[] {
  const minCount = options?.minCount ?? 1;
  const idx = ids.indexOf(id);
  if (idx >= 0) {
    if (ids.length <= minCount) return [...ids];
    return ids.filter((x) => x !== id);
  }
  return [...ids, id];
}
