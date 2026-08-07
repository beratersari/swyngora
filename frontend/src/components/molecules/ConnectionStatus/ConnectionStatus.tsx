import { Dot, Label, Pill } from './ConnectionStatus.styles';
import type { ConnectionStatusProps } from './ConnectionStatus.types';

/** Presentational API/live indicator for desk chrome. */
export function ConnectionStatus({ status, label }: ConnectionStatusProps) {
  return (
    <Pill role="status" aria-live="polite">
      <Dot $status={status} aria-hidden />
      <Label>{label}</Label>
    </Pill>
  );
}
