import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { CoinSituation } from '@/components/molecules/CoinSituation';
import { useGetMarketSituationQuery } from '@/libs/api';
import { useDocumentVisible } from '@/libs/hooks';
import { OUTLIER_SITUATION_POLL_MS } from './constants';
import type { OutlierSituationProps } from './OutlierSituation.types';

/** The coin-page situation, loaded for one opened outlier row. */
export function OutlierSituation({ symbol }: OutlierSituationProps) {
  const { t } = useTranslation('detail');
  const visible = useDocumentVisible();
  const query = useGetMarketSituationQuery(
    { symbol },
    {
      skip: symbol.trim() === '',
      pollingInterval: visible ? OUTLIER_SITUATION_POLL_MS : 0,
      refetchOnFocus: true,
    },
  );
  return (
    <div data-testid="outlier-situation" onClick={(event) => event.stopPropagation()}>
      <Text variant="caption" color="secondary">
        {t('situation.deskLead')}
      </Text>
      <CoinSituation
        report={query.data}
        isLoading={query.isLoading && query.data == null}
        error={query.isError && query.data == null ? t('situation.deskFailed') : undefined}
      />
    </div>
  );
}
