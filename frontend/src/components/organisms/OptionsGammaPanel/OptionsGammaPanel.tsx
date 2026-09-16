import { useTranslation } from 'react-i18next';
import { Skeleton } from '@/components/atoms/Skeleton';
import { Text } from '@/components/atoms/Text';
import { formatPrice } from '@/libs/utils';
import {
  formatFlipDist,
  formatGex,
  gexBarPct,
  gammaHasData,
  isNearFlip,
  isNearSpot,
  maxAbsGex,
} from './helpers';
import { Axis, Bar, Col, Down, Head, HeadCard, Profile, Stack, Up } from './OptionsGammaPanel.styles';
import type { OptionsGammaPanelProps } from './OptionsGammaPanel.types';

export function OptionsGammaPanel({ data, isLoading }: OptionsGammaPanelProps) {
  const { t } = useTranslation('options');
  const gamma = data?.gamma;
  const profile = gamma?.profile ?? [];
  const peak = maxAbsGex(profile);

  if (isLoading && !data) {
    return <Skeleton height={220} />;
  }
  if (!gammaHasData(gamma)) {
    return (
      <Text variant="bodySm" color="secondary">
        {t('gamma.empty')}
      </Text>
    );
  }

  const regime = gamma?.regime;
  const flipDist = formatFlipDist(gamma?.flipDistPct);

  return (
    <Stack>
      <Head>
        <HeadCard>
          <Text variant="caption" color="secondary">
            {t('gamma.net')}
          </Text>
          <Text variant="h3" color="primary">
            {formatGex(gamma?.netUsd)}
          </Text>
          <Text variant="caption" color="secondary">
            {t('gamma.netHint', { call: formatGex(gamma?.callUsd), put: formatGex(gamma?.putUsd) })}
          </Text>
        </HeadCard>
        <HeadCard>
          <Text variant="caption" color="secondary">
            {t('gamma.regime')}
          </Text>
          <Text variant="h3" color="primary">
            {regime ? t(`gamma.regimeLabel.${regime}` as 'gamma.regimeLabel.flat') : '—'}
          </Text>
          <Text variant="caption" color="secondary">
            {regime ? t(`gamma.regimeHint.${regime}` as 'gamma.regimeHint.flat') : t('gamma.needIv')}
          </Text>
        </HeadCard>
        <HeadCard>
          <Text variant="caption" color="secondary">
            {t('gamma.flip')}
          </Text>
          <Text variant="h3" color="primary">
            {gamma?.flip ? formatPrice(gamma.flip) : '—'}
          </Text>
          <Text variant="caption" color="secondary">
            {gamma?.flip && gamma.flipSide
              ? t(`gamma.flipHint.${gamma.flipSide}` as 'gamma.flipHint.at', {
                  spot: data?.indexPrice ? formatPrice(data.indexPrice) : '—',
                  dist: flipDist ?? '',
                })
              : t('gamma.flipEmpty')}
          </Text>
        </HeadCard>
      </Head>

      {profile.length > 0 ? (
        <div>
          <Text variant="bodySm" color="primary">
            {t('gamma.profileTitle')}
          </Text>
          <Text variant="caption" color="secondary">
            {t('gamma.profileLegend')}
          </Text>
          <Profile aria-label={t('gamma.profileAria')}>
            {profile.map((row) => {
              const net = Number(row.netUsd);
              const long = Number.isFinite(net) && net >= 0;
              const pct = gexBarPct(row.netUsd, peak);
              return (
                <Col
                  key={row.spot}
                  $spot={isNearSpot(row.spot, data?.indexPrice)}
                  $flip={isNearFlip(row.spot, gamma?.flip)}
                  title={`${formatPrice(row.spot)} · ${formatGex(row.netUsd)}`}
                >
                  <Up>{long ? <Bar $height={pct} $side="long" /> : null}</Up>
                  <Axis />
                  <Down>{long ? null : <Bar $height={pct} $side="short" />}</Down>
                </Col>
              );
            })}
          </Profile>
        </div>
      ) : null}
    </Stack>
  );
}
