import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import { useDisplayCurrency } from '@/libs/hooks';
import { longShare, orderedCardWindows } from './LiquidationWindowCards.helpers';
import {
  CardButton,
  CardHead,
  Grid,
  SideLabel,
  SideRow,
  SideValue,
  SplitBar,
  SplitLong,
  SplitShort,
  TotalValue,
  WindowLabel,
} from './LiquidationWindowCards.styles';
import type { LiquidationWindowCardsProps } from './LiquidationWindowCards.types';

/**
 * CoinGlass-style time-window cards: total on top, then long, then short.
 */
export function LiquidationWindowCards({
  windows,
  selectedWindow,
  onSelect,
  isLoading,
}: LiquidationWindowCardsProps) {
  const { t } = useTranslation('liquidations');
  const { formatCompact } = useDisplayCurrency();
  const rows = orderedCardWindows(windows);

  if (isLoading && windows.length === 0) {
    return (
      <Grid>
        {rows.map((row) => (
          <Skeleton key={row.window} height={132} />
        ))}
      </Grid>
    );
  }

  return (
    <Grid>
      {rows.map((row) => {
        const id = row.window ?? '24h';
        const active = id === selectedWindow;
        const share = longShare(row);
        const longPct = Math.round(share * 100);
        return (
          <CardButton
            key={id}
            type="button"
            $active={active}
            aria-pressed={active}
            aria-label={t('cards.selectAria', { window: t(`windows.${id}` as 'windows.1h') })}
            onClick={() => onSelect(id)}
          >
            <CardHead>
              <WindowLabel>{t(`windows.${id}` as 'windows.1h')}</WindowLabel>
              {row.complete === false ? (
                <Text variant="caption" color="secondary">
                  {t('status.collecting')}
                </Text>
              ) : null}
            </CardHead>
            <TotalValue>{formatCompact(row.totalNotional, 'USDT')}</TotalValue>
            <Text variant="caption" color="secondary">
              {t('cards.total')}
              {row.count != null ? ` · ${t('cards.count', { count: row.count })}` : ''}
            </Text>
            <SideRow>
              <SideLabel $tone="long">{t('cards.long')}</SideLabel>
              <SideValue $tone="long">{formatCompact(row.longNotional, 'USDT')}</SideValue>
            </SideRow>
            <SideRow>
              <SideLabel $tone="short">{t('cards.short')}</SideLabel>
              <SideValue $tone="short">{formatCompact(row.shortNotional, 'USDT')}</SideValue>
            </SideRow>
            <SplitBar aria-hidden>
              <SplitLong $pct={longPct} />
              <SplitShort $pct={Math.max(0, 100 - longPct)} />
            </SplitBar>
          </CardButton>
        );
      })}
    </Grid>
  );
}
