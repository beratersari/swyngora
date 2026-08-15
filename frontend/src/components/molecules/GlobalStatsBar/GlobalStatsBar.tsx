import { useTranslation } from 'react-i18next';
import { Bar, Stat, StatLabel, StatValue } from './GlobalStatsBar.styles';
import type { GlobalStatsBarProps } from './GlobalStatsBar.types';

/** Compact market snapshot under the header (CoinMarketCap-style). */
export function GlobalStatsBar({
  coinCount,
  volumeLabel,
  btcPrice,
  btcChange,
  btcUp,
}: GlobalStatsBarProps) {
  const { t } = useTranslation('common');
  return (
    <Bar role="region" aria-label={t('stats.aria', { defaultValue: 'Market snapshot' })}>
      <Stat>
        <StatLabel>{t('stats.cryptos', { defaultValue: 'Cryptos' })}:</StatLabel>
        <StatValue $tone="accent">{coinCount.toLocaleString()}</StatValue>
      </Stat>
      <Stat>
        <StatLabel>{t('stats.volume', { defaultValue: '24h Vol' })}:</StatLabel>
        <StatValue $tone="accent">{volumeLabel}</StatValue>
      </Stat>
      {btcPrice ? (
        <Stat>
          <StatLabel>BTC:</StatLabel>
          <StatValue $tone="accent">{btcPrice}</StatValue>
          {btcChange ? <StatValue $tone={btcUp ? 'up' : 'down'}>{btcChange}</StatValue> : null}
        </Stat>
      ) : null}
    </Bar>
  );
}
