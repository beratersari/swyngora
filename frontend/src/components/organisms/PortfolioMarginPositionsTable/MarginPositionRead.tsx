import { useTranslation } from 'react-i18next';
import { isCashEquity, PositionSituation } from '@/components/molecules/PositionSituation';
import { useGetMarketSituationQuery, type MarginPosition } from '@/libs/api';

/** Same cadence as the spot position read. */
const MARGIN_SITUATION_POLL_MS = 60_000;

export function MarginPositionRead({ row }: { row: MarginPosition }) {
  const { t } = useTranslation('detail');
  const equity = isCashEquity(row.exchange);
  const query = useGetMarketSituationQuery(
    { symbol: row.symbol ?? '' },
    {
      skip: equity || !row.symbol,
      pollingInterval: MARGIN_SITUATION_POLL_MS,
      refetchOnFocus: true,
    },
  );
  if (equity) return null;
  const sideKey = row.side === 'short' ? 'short' : 'long';
  return (
    <PositionSituation
      lead={t('situation.marginLead', {
        side: t(`situation.marginSide.${sideKey}`),
        leverage: row.leverage ?? '',
      })}
      entry={row.entrySituation}
      now={query.data}
      nowLoading={query.isLoading && query.data == null}
    />
  );
}
