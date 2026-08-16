import { Button, Skeleton } from 'antd';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import { useTranslation } from 'react-i18next';
import { BrandTag } from '@/components/atoms/BrandTag';
import { Text } from '@/components/atoms/Text';
import { formatExactDateTime, formatPrice, formatSymbolDisplay } from '@/libs/utils';
import {
  Card,
  CardFooter,
  CardTop,
  FactorRow,
  Grid,
  LevelLabel,
  LevelRow,
  Levels,
  LevelValue,
  PairBlock,
} from './SwingEngineGrid.styles';
import type { SwingEngineGridProps } from './SwingEngineGrid.types';

function gradeVariant(grade?: string): 'gradeA' | 'gradeB' | 'gradeC' {
  if (grade === 'A') return 'gradeA';
  if (grade === 'B') return 'gradeB';
  return 'gradeC';
}

export function SwingEngineGrid({
  items,
  loading = false,
  emptyText,
  onOpen,
}: SwingEngineGridProps) {
  const { t, i18n } = useTranslation('signals');

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
    return <DeskEmpty title={emptyText ?? t('engine.empty')} />;
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
              <BrandTag variant="outline">{(s.setupType ?? '').replace(/_/g, ' ')}</BrandTag>
              <BrandTag variant="outline">{t('engine.regime', { regime: s.btcRegime ?? 'unknown' })}</BrandTag>
              {s.fresh ? <BrandTag variant="live">{t('engine.fresh')}</BrandTag> : null}
            </FactorRow>
            {s.levels ? (
              <Levels>
                <LevelRow>
                  <LevelLabel>{t('engine.entry')}</LevelLabel>
                  <LevelValue>{formatPrice(s.levels.entry)}</LevelValue>
                </LevelRow>
                <LevelRow>
                  <LevelLabel>{t('engine.stop')}</LevelLabel>
                  <LevelValue>{formatPrice(s.levels.stopLoss)}</LevelValue>
                </LevelRow>
                <LevelRow>
                  <LevelLabel>{t('engine.target')}</LevelLabel>
                  <LevelValue>{formatPrice(s.levels.takeProfit)}</LevelValue>
                </LevelRow>
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
                {s.barTime
                  ? ` · ${t('engine.triggered', { date: formatExactDateTime(s.barTime, i18n.language) })}`
                  : ''}
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
