import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import { formatIvPct, formatSkewPts, ivBarPct, maxAtmIv, maxCallPutIv } from './helpers';
import {
  Chart,
  Charts,
  Fill,
  Head,
  HeadCard,
  Pair,
  PairBar,
  Row,
  Stack,
  Track,
} from './OptionsVolStructure.styles';
import type { OptionsVolStructureProps } from './OptionsVolStructure.types';

export function OptionsVolStructure({ data, isLoading }: OptionsVolStructureProps) {
  const { t } = useTranslation('options');
  const points = data?.termStructure ?? [];
  const skew = data?.skew;
  const atmMax = maxAtmIv(points);
  const pairMax = maxCallPutIv(points);

  if (isLoading && !data) {
    return <Skeleton height={220} />;
  }

  if (points.length === 0 && !skew) {
    return (
      <Text variant="bodySm" color="secondary">
        {t('vol.empty')}
      </Text>
    );
  }

  return (
    <Stack>
      <Head>
        <HeadCard>
          <Text variant="caption" color="secondary">
            {t('vol.term')}
          </Text>
          <Text variant="h3" color="primary">
            {formatSkewPts(skew?.termSlope)}
          </Text>
          <Text variant="caption" color="secondary">
            {skew?.frontDte != null && skew?.backDte != null
              ? t('vol.termHint', { front: skew.frontDte, back: skew.backDte })
              : t('vol.termEmpty')}
          </Text>
        </HeadCard>
        <HeadCard>
          <Text variant="caption" color="secondary">
            {t('vol.atmSkew')}
          </Text>
          <Text variant="h3" color="primary">
            {formatSkewPts(skew?.atmSkew)}
          </Text>
          <Text variant="caption" color="secondary">
            {skew?.lean ? t(`vol.lean.${skew.lean}` as 'vol.lean.balanced') : t('vol.skewHint')}
          </Text>
        </HeadCard>
        <HeadCard>
          <Text variant="caption" color="secondary">
            {t('vol.wing')}
          </Text>
          <Text variant="h3" color="primary">
            {formatSkewPts(skew?.wingSkew)}
          </Text>
          <Text variant="caption" color="secondary">
            {t('vol.wingHint')}
          </Text>
        </HeadCard>
      </Head>

      <Charts>
        <Chart>
          <Text variant="bodySm" color="primary">
            {t('vol.termTitle')}
          </Text>
          <Text variant="caption" color="secondary">
            {t('vol.termLegend')}
          </Text>
          {points.map((p) => (
            <Row key={p.expiry ?? p.label}>
              <Text variant="caption" color="secondary">
                {p.label ?? '—'}
              </Text>
              <Track title={formatIvPct(p.atmIv)}>
                <Fill $width={ivBarPct(p.atmIv, atmMax)} />
              </Track>
              <Text variant="caption" color="primary">
                {formatIvPct(p.atmIv)}
              </Text>
            </Row>
          ))}
        </Chart>
        <Chart>
          <Text variant="bodySm" color="primary">
            {t('vol.skewTitle')}
          </Text>
          <Text variant="caption" color="secondary">
            {t('vol.skewLegend')}
          </Text>
          {points.map((p) => (
            <Row key={`sk-${p.expiry ?? p.label}`}>
              <Text variant="caption" color="secondary">
                {p.label ?? '—'}
              </Text>
              <Pair aria-label={t('vol.pairAria', { label: p.label ?? '' })}>
                <PairBar $side="call" $height={ivBarPct(p.callIv, pairMax)} />
                <PairBar $side="put" $height={ivBarPct(p.putIv, pairMax)} />
              </Pair>
              <Text variant="caption" color="primary">
                {formatSkewPts(p.skew)}
              </Text>
            </Row>
          ))}
        </Chart>
      </Charts>
    </Stack>
  );
}
