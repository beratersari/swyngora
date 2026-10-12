import { FillMoves } from '@/components/molecules/FillMoves';
import { PositionSituation, isCashEquity } from '@/components/molecules/PositionSituation';
import { useGetMarketSituationQuery, type SpotPosition } from '@/libs/api';
import { POSITION_SITUATION_POLL_MS } from './constants';

export function PositionRead({ row }: { row: SpotPosition }) {
  const equity = isCashEquity(row.exchange);
  const query = useGetMarketSituationQuery(
    { symbol: row.symbol ?? '' },
    {
      skip: equity || !row.symbol,
      pollingInterval: POSITION_SITUATION_POLL_MS,
      refetchOnFocus: true,
    },
  );
  return (
    <>
      <FillMoves moves={row.moves} />
      {equity ? null : (
        <PositionSituation
          entry={row.entrySituation}
          now={query.data}
          nowLoading={query.isLoading && query.data == null}
        />
      )}
    </>
  );
}
