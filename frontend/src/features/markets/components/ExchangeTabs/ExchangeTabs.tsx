import { Skeleton } from '@/components/atoms/Skeleton';
import type { MarketExchange } from '@/libs/api';
import { EXCHANGE_LABELS } from './ExchangeTabs.constants';
import { StyledTabs } from './ExchangeTabs.styles';
import type { ExchangeTabsProps } from './ExchangeTabs.types';

export function ExchangeTabs({ exchanges, value, onChange, isLoading }: ExchangeTabsProps) {
  if (isLoading) {
    return (
      <Skeleton variant="button" width={280} height={40} active aria-label="Loading exchanges" />
    );
  }

  const items = (exchanges.length ? exchanges : [value]).map((id) => ({
    key: id,
    label: EXCHANGE_LABELS[id] ?? id,
  }));

  return (
    <StyledTabs
      activeKey={value}
      items={items}
      onChange={(key) => onChange(key as MarketExchange)}
    />
  );
}
