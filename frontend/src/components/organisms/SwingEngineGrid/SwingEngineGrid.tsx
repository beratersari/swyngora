import { Button, Empty, Skeleton } from 'antd';
import { useTranslation } from 'react-i18next';
import { BrandTag } from '@/components/atoms/BrandTag';
import { Text } from '@/components/atoms/Text';
import { formatPrice, formatSymbolDisplay } from '@/libs/utils';
import {
  Card,
  CardFooter,
  CardTop,
  FactorRow,
  Grid,
  Levels,
  PairBlock,
} from './SwingEngineGrid.styles';
import type { SwingEngineGridProps } from './SwingEngineGrid.types';

function gradeVariant(grade?: string): 'live' | 'status' | 'paused' {
  if (grade === 'A') return 'live';
  if (grade === 'B') return 'status';
  return 'paused';
}

export function SwingEngineGrid({
  items,
  loading = false,
  emptyText,
  onOpen,
}: SwingEngineGridProps) {
  const { t } = useTranslation('signals');

  if (loading && items.length === 0) {
    return (
      <Grid>
        {[0, 1, 2].map((i) => (
          <Card key={i}>
            <Skeleton active paragraph={{ rows: 4 }} />
          </Card>
        ))}
      </Grid>
    );
  }

  if (items.length === 0) {
    return <Empty description={emptyText ?? t('engine.empty')} />;
  }

  return (
    <Grid>
      {items.map((s) => {
        const key = `${s.exchange}|${s.symbol}|${s.barTime ?? s.stage}`;
        return (
          <Card key={key}>
            <CardTop>
              <PairBlock>
                <Text variant="label" mono color="primary">
                  {formatSymbolDisplay(s.symbol)}
                </Text>
                <BrandTag variant="exchange">{s.exchange}</BrandTag>
              </PairBlock>
              <BrandTag variant={gradeVariant(s.grade)}>
                {t('engine.grade', { grade: s.grade ?? 'C', score: s.swingScore ?? 0 })}
              </BrandTag>
            </CardTop>
            <FactorRow>
              <BrandTag variant={s.accepted ? 'live' : 'paused'}>{s.stage}</BrandTag>
              <BrandTag variant="status">{(s.setupType ?? '').replace(/_/g, ' ')}</BrandTag>
              <BrandTag variant="status">{t('engine.regime', { regime: s.btcRegime ?? 'unknown' })}</BrandTag>
              {s.fresh ? <BrandTag variant="live">{t('engine.fresh')}</BrandTag> : null}
            </FactorRow>
            {s.levels ? (
              <Levels>
                <div>
                  <Text variant="caption" color="secondary">
                    {t('engine.entry')}
                  </Text>
                  <Text variant="numeric" color="primary">
                    {formatPrice(s.levels.entry)}
                  </Text>
                </div>
                <div>
                  <Text variant="caption" color="secondary">
                    {t('engine.stop')}
                  </Text>
                  <Text variant="numeric" color="primary">
                    {formatPrice(s.levels.stopLoss)}
                  </Text>
                </div>
                <div>
                  <Text variant="caption" color="secondary">
                    {t('engine.target')}
                  </Text>
                  <Text variant="numeric" color="primary">
                    {formatPrice(s.levels.takeProfit)}
                  </Text>
                </div>
              </Levels>
            ) : null}
            <Text variant="caption" color="secondary">
              {s.levels
                ? t('engine.rr', { rr: (s.levels.rr ?? 0).toFixed(2), risk: (s.levels.riskPct ?? 0).toFixed(2) })
                : (s.reasons ?? []).slice(0, 2).join(' · ')}
            </Text>
            <CardFooter>
              <Text variant="caption" color="secondary">
                {s.interval} · RSI {s.rsi != null ? s.rsi.toFixed(1) : '—'}
              </Text>
              {onOpen && s.exchange && s.symbol ? (
                <Button size="small" type="link" onClick={() => onOpen(s.exchange!, s.symbol!)}>
                  {t('openChart')}
                </Button>
              ) : null}
            </CardFooter>
          </Card>
        );
      })}
    </Grid>
  );
}
