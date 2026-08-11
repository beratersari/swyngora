import { Text } from '@/components/atoms/Text';
import { Mark, Wrap } from './DeskEmpty.styles';
import type { DeskEmptyProps } from './DeskEmpty.types';

/** Branded empty state — no Ant grey illustration. */
export function DeskEmpty({ title, hint, extra, className }: DeskEmptyProps) {
  return (
    <Wrap className={className} role="status">
      <Mark aria-hidden />
      <Text variant="body" color="primary">
        {title}
      </Text>
      {hint ? (
        <Text variant="caption" color="secondary">
          {hint}
        </Text>
      ) : null}
      {extra}
    </Wrap>
  );
}
