import { Alert } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { rtkErrorMessage, useGetBookGapsQuery } from '@/libs/api';
import { useDocumentVisible } from '@/libs/hooks';
import { BOOK_GAPS_POLL_MS } from './constants';
import { money, pct } from './helpers';
import { Card, GapList, Stats } from './BookGapsCard.styles';
import type { BookGapsCardProps } from './BookGapsCard.types';

export function BookGapsCard({ exchange, symbol }: BookGapsCardProps) {
  const { t } = useTranslation('gaps');
  const visible = useDocumentVisible();
  const query = useGetBookGapsQuery(
    { exchange, symbol, rangePct: 2 },
    { skip: !symbol || !exchange, pollingInterval: visible ? BOOK_GAPS_POLL_MS : 0 },
  );
  const data = query.data;
  const easier =
    data?.easier === 'up' ? t('easierUp') : data?.easier === 'down' ? t('easierDown') : t('easierEven');

  return (
    <Card data-testid="book-gaps" aria-label={t('title')}>
      <Text variant="label" color="primary">
        {t('title')}
      </Text>
      <Text variant="caption" color="secondary">
        {t('subtitle')}
      </Text>
      {query.isError ? (
        <Alert
          type="error"
          showIcon
          message={t('loadFailed')}
          description={rtkErrorMessage(query.error, { resource: t('resource') })}
        />
      ) : null}
      {data ? (
        <>
          <Text variant="body" color="primary">
            {easier}
          </Text>
          <Stats>
            <Text variant="caption" color="secondary">
              {t('upSize', { size: money(data.upNotional), empty: pct(data.upGapPct) })}
            </Text>
            <Text variant="caption" color="secondary">
              {t('downSize', { size: money(data.downNotional), empty: pct(data.downGapPct) })}
            </Text>
          </Stats>
          {data.summary ? (
            <Text variant="caption" color="secondary">
              {data.summary}
            </Text>
          ) : null}
          <GapList>
            {(data.gaps ?? []).map((gap) => (
              <li key={`${gap.direction}-${gap.fromPrice}-${gap.toPrice}`}>
                <Text variant="numeric" color="primary">
                  {t('gapRow', {
                    way: gap.direction === 'down' ? t('down') : t('up'),
                    from: gap.fromPrice,
                    to: gap.toPrice,
                    width: pct(gap.widthPct),
                    size: money(gap.notional),
                  })}
                </Text>
              </li>
            ))}
          </GapList>
          {(data.gaps ?? []).length === 0 ? (
            <Text variant="caption" color="secondary">
              {t('noGaps')}
            </Text>
          ) : null}
        </>
      ) : null}
    </Card>
  );
}
