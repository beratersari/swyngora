import { ChipGroup } from '@/components/molecules/chip-group';
import type { ExchangeChipsProps } from './ExchangeChips.types';

export function ExchangeChips({
  exchanges,
  selected,
  onSelect,
  isLoading,
}: ExchangeChipsProps) {
  return (
    <ChipGroup
      options={exchanges.map((e) => ({ value: e, label: e }))}
      selected={selected}
      onSelect={onSelect}
      mode="single"
      shape="pill"
      horizontalScroll
      isLoading={isLoading}
    />
  );
}
