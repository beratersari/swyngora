import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import { formatPrice } from '@/libs/utils';
import { Card, Strip } from './PortfolioSummaryStrip.styles';
import type { PortfolioSummaryStripProps } from './PortfolioSummaryStrip.types';

function pnlColor(n: number | undefined): 'primary' | 'success' | 'error' {
  if (n == null || !Number.isFinite(n) || Math.abs(n) < 1e-12) return 'primary';
  return n > 0 ? 'success' : 'error';
}

export function PortfolioSummaryStrip({ view, isLoading, currency = 'USDT' }: PortfolioSummaryStripProps) {
  const { t } = useTranslation('portfolio');
  if (isLoading && !view) {
    return <Skeleton height={88} />;
  }
  const items: { key: string; label: string; value: string; color?: 'primary' | 'success' | 'error' }[] = [
    { key: 'equity', label: t('metrics.equity'), value: `${formatPrice(view?.equity)} ${currency}` },
    { key: 'cash', label: t('metrics.cash'), value: `${formatPrice(view?.cashBalance)} ${currency}` },
    { key: 'available', label: t('metrics.available'), value: `${formatPrice(view?.availableCash)} ${currency}` },
    { key: 'pos', label: t('metrics.positions'), value: `${formatPrice(view?.positionsValue)} ${currency}` },
    {
      key: 'upnl',
      label: t('metrics.unrealized'),
      value: `${formatPrice(view?.unrealizedPnL)} ${currency}`,
      color: pnlColor(view?.unrealizedPnL),
    },
    {
      key: 'rpnl',
      label: t('metrics.realized'),
      value: `${formatPrice(view?.realizedPnLTotal)} ${currency}`,
      color: pnlColor(view?.realizedPnLTotal),
    },
    {
      key: 'tpnl',
      label: t('metrics.totalPnl'),
      value: `${formatPrice(view?.totalPnL)} ${currency}`,
      color: pnlColor(view?.totalPnL),
    },
  ];
  return (
    <Strip>
      {items.map((it) => (
        <Card key={it.key}>
          <Text variant="caption" color="secondary">
            {it.label}
          </Text>
          <Text variant="label" mono color={it.color ?? 'primary'}>
            {it.value}
          </Text>
        </Card>
      ))}
    </Strip>
  );
}
