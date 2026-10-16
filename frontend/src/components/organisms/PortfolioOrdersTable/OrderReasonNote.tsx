import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import {
  compareCoinSituation,
  situationChangeLines,
} from '@/components/molecules/CoinSituation';
import { useGetMarketSituationQuery, type CoinSituation } from '@/libs/api';

type OrderReasonNoteProps = {
  symbol: string;
  entry: CoinSituation;
};

/** Says when the live read has left the clauses stored on an open buy. */
export function OrderReasonNote({ symbol, entry }: OrderReasonNoteProps) {
  const { t } = useTranslation(['portfolio', 'detail']);
  const query = useGetMarketSituationQuery({ symbol }, { skip: !symbol });
  if (!query.isSuccess) return null;
  const comparison = compareCoinSituation(entry, query.data);
  if (!comparison.clauses?.length) return null;
  const lines = situationChangeLines(
    entry,
    query.data,
    comparison,
    'live',
    t as unknown as Parameters<typeof situationChangeLines>[4],
  );
  return (
    <Text variant="caption" color="secondary">
      {t('portfolio:orders.reasonLeft', {
        defaultValue: 'The market has left the reason on this order.',
      })}
      {lines.length > 0 ? ` ${lines.join(' ')}` : ''}
    </Text>
  );
}
