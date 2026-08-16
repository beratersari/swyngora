import { Button, Skeleton } from 'antd';
import { DeskEmpty } from '@/components/molecules/DeskEmpty';
import { useTranslation } from 'react-i18next';
import { BrandTag } from '@/components/atoms/BrandTag';
import { Text } from '@/components/atoms/Text';
import { formatExactDateTime, formatSymbolDisplay, ruleTypeShort } from '@/libs/utils';
import {
  Card,
  CardFooter,
  CardTop,
  FactorRow,
  Grid,
  PairBlock,
  SummaryList,
} from './SignalsSetupGrid.styles';
import type { SignalsSetupGridProps } from './SignalsSetupGrid.types';

export function SignalsSetupGrid({
  setups,
  loading = false,
  emptyText,
  onOpen,
}: SignalsSetupGridProps) {
  const { t, i18n } = useTranslation('signals');

  if (loading && setups.length === 0) {
    return (
      <Grid>
        {[0, 1, 2].map((i) => (
          <Card key={i}>
            <Skeleton active paragraph={{ rows: 3 }} />
          </Card>
        ))}
      </Grid>
    );
  }

  if (setups.length === 0) {
    return <DeskEmpty title={emptyText ?? t('setups.empty')} />;
  }

  return (
    <Grid>
      {setups.map((s) => (
        <Card key={s.key}>
          <CardTop>
            <PairBlock>
              <Text variant="label" mono color="primary">
                {formatSymbolDisplay(s.symbol)}
              </Text>
              <BrandTag variant="exchange">{s.exchange}</BrandTag>
            </PairBlock>
            <BrandTag variant={s.grade === 'A' ? 'live' : 'status'}>
              {t('setups.grade', { grade: s.grade, score: s.score })}
            </BrandTag>
          </CardTop>
          <FactorRow>
            {s.factors.map((f) => (
              <BrandTag key={f} variant="status">
                {ruleTypeShort(f)}
              </BrandTag>
            ))}
            <BrandTag variant={s.sameBar ? 'live' : 'paused'}>
              {s.sameBar ? t('setups.sameBar') : t('setups.window')}
            </BrandTag>
            <Text variant="caption" color="secondary">
              {s.interval}
            </Text>
          </FactorRow>
          <SummaryList>
            {s.summaries.slice(0, 3).map((line) => (
              <li key={line}>
                <Text variant="caption" color="secondary">
                  {line}
                </Text>
              </li>
            ))}
          </SummaryList>
          <CardFooter>
            <Text variant="caption" color="secondary">
              {formatExactDateTime(s.latestAt, i18n.language)}
            </Text>
            {onOpen ? (
              <Button size="small" type="link" onClick={() => onOpen(s.exchange, s.symbol)}>
                {t('openChart')}
              </Button>
            ) : null}
          </CardFooter>
        </Card>
      ))}
    </Grid>
  );
}
