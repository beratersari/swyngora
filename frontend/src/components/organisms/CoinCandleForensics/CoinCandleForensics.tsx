import { Button } from 'antd';
import { Link } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { ForensicsSimilar } from '@/components/organisms/ForensicsSimilar';
import { ForensicsTimeline } from '@/components/organisms/ForensicsTimeline';
import { ForensicsWhy } from '@/components/organisms/ForensicsWhy';
import { Actions, Card, Header } from './CoinCandleForensics.styles';
import type { CoinCandleForensicsProps } from './CoinCandleForensics.types';

export function CoinCandleForensics({
  timeLabel,
  report,
  isLoading,
  error,
  fullReadTo,
  onClear,
  onOpenMove,
}: CoinCandleForensicsProps) {
  const { t } = useTranslation('detail');
  if (!timeLabel) {
    return (
      <Text variant="caption" color="secondary" data-testid="chart-forensics-hint">
        {t('chart.forensicsHint')}
      </Text>
    );
  }
  const why = report?.why;
  const showNote = !isLoading && !error && Boolean(report?.note) && !why?.headline && !why?.before && !why?.during;
  return (
    <Card aria-label={t('chart.forensicsTitle')} data-testid="chart-forensics">
      <Header>
        <Text variant="label">{t('chart.forensicsTitle')}</Text>
        <Text variant="caption" color="secondary">
          {t('chart.forensicsSince', { time: timeLabel })}
        </Text>
        <Actions>
          {fullReadTo ? (
            <Link to={fullReadTo} data-testid="chart-forensics-full">
              {t('chart.forensicsFull')}
            </Link>
          ) : null}
          <Button type="link" size="small" onClick={onClear} data-testid="chart-forensics-clear">
            {t('chart.cvdClickClear')}
          </Button>
        </Actions>
      </Header>
      <Text variant="caption" color="secondary">
        {t('chart.forensicsWindow')}
      </Text>
      {isLoading ? (
        <Text variant="caption" color="secondary" data-testid="chart-forensics-loading">
          {t('chart.forensicsLoading')}
        </Text>
      ) : error ? (
        <Text variant="bodySm" data-testid="chart-forensics-error">
          {error}
        </Text>
      ) : (
        <>
          <ForensicsWhy why={why} />
          <ForensicsTimeline timeline={report?.timeline} />
          {showNote ? (
            <Text variant="caption" color="secondary">
              {report?.note}
            </Text>
          ) : null}
          <ForensicsSimilar
            similar={report?.similar}
            similarError={report?.similarError}
            onOpenMove={onOpenMove}
          />
        </>
      )}
    </Card>
  );
}
