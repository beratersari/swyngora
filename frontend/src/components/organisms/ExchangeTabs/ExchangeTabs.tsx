import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { TabBtn, TabList } from './ExchangeTabs.styles';
import type { ExchangeTabsProps } from './ExchangeTabs.types';

export function ExchangeTabs({ exchanges, value, onChange, isLoading }: ExchangeTabsProps) {
  const { t } = useTranslation('common');

  if (isLoading && exchanges.length === 0) {
    return (
      <Skeleton variant="button" width={280} height={40} active aria-label={t('a11y.loadingExchanges')} />
    );
  }

  return (
    <TabList role="tablist" aria-label={t('a11y.exchangeTabs')}>
      {exchanges.map((ex) => (
        <TabBtn
          key={ex}
          type="button"
          role="tab"
          aria-selected={ex === value}
          $active={ex === value}
          onClick={() => onChange(ex as ExchangeTabsProps['value'])}
        >
          {t(`exchanges.${ex}`, { defaultValue: ex })}
        </TabBtn>
      ))}
    </TabList>
  );
}
