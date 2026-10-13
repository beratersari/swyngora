export function situationLeaveGroupKey(clause: string, kind: string, group: 'still' | 'left'): string {
  return `${clause}:${kind}:${group}`;
}
