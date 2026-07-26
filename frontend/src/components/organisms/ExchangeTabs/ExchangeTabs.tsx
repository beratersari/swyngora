import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { StyledTabs } from './ExchangeTabs.styles';
import type { ExchangeTabsProps } from './ExchangeTabs.types';

export function ExchangeTabs({ exchanges, value, onChange, isLoading }: ExchangeTabsProps) {
  const { t } = useTranslation('common');

  if (isLoading && exchanges.length === 0) {
    return (
      <Skeleton variant="button" width={280} height={40} active aria-label={t('a11y.loadingExchanges')} />
    );
  }

  return (
    <StyledTabs
      activeKey={value}
      onChange={(key) => onChange(key as ExchangeTabsProps['value'])}
      items={exchanges.map((ex) => ({
        key: ex,
        label: ex,
      }))}
    />
  );
}
