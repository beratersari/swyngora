import { Fragment } from 'react';
import { Alert } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { formatPrice } from '@/libs/utils/formatPrice';
import type { PreviewWarningId } from './constants';
import { formatPreviewPct, previewRows, previewWarnings } from './helpers';
import { Grid, Label, Panel, Value } from './OrderPreview.styles';
import type { OrderPreviewProps } from './OrderPreview.types';

function warningTone(id: PreviewWarningId): 'info' | 'error' | 'warning' {
  if (id === 'dailyLoss') return 'info';
  if (id === 'blocked') return 'error';
  return 'warning';
}

/**
 * Cost and risk of the ticket in front of the user. The page supplies the preview.
 * This does not place an order.
 */
export function OrderPreview({ preview, loading, failed }: OrderPreviewProps) {
  const { t } = useTranslation('portfolio');
  if (!preview && !loading && !failed) return null;

  const rows = preview ? previewRows(preview) : [];
  const warnings = preview ? previewWarnings(preview) : [];
  const weight = preview?.risk?.projectedWeightPct;
  const exposure = preview?.risk?.projectedExposurePct;
  const asset = preview?.risk?.asset;

  return (
    <Panel role="status" aria-live="polite" aria-label={t('preview.title')}>
      <Text variant="caption" color="primary">
        {t('preview.title')}
      </Text>
      {loading && !preview ? (
        <Text variant="caption" color="secondary">
          {t('preview.pricing')}
        </Text>
      ) : null}
      {failed && !preview ? <Alert type="warning" showIcon message={t('preview.failed')} /> : null}
      {preview ? (
        <>
          <Text variant="caption" color="secondary">
            {preview.routed
              ? `${t('preview.fromRoute')} `
              : preview.bookSource === 'book'
                ? `${t('preview.fromBook')} `
                : ''}
            {preview.fillsNow ? t('preview.fillsNow') : t('preview.rests')}
            {preview.product === 'margin' && (preview.leverage ?? 0) > 0
              ? ` · ${t('preview.leverage', { lev: preview.leverage })}`
              : ''}
          </Text>
          <Grid>
            {rows.map((row) => (
              <Fragment key={row.id}>
                <Label>
                  <Text variant="caption" color="secondary">
                    {t(`preview.${row.labelKey}`)}
                  </Text>
                </Label>
                <Value>
                  <Text variant="caption" color="primary">
                    {row.value}
                  </Text>
                </Value>
              </Fragment>
            ))}
          </Grid>
          {weight != null && asset ? (
            <Text variant="caption" color="secondary">
              {t('preview.weight', { asset, pct: formatPreviewPct(weight) })}
            </Text>
          ) : null}
          {exposure != null ? (
            <Text variant="caption" color="secondary">
              {t('preview.exposure', { pct: formatPreviewPct(exposure) })}
            </Text>
          ) : null}
          {preview.sizing ? (
            <Text variant="caption" color="secondary">
              {t('preview.sized', {
                pct: formatPreviewPct(preview.sizing.riskPercent),
                budget: formatPrice(preview.sizing.riskBudget),
              })}
            </Text>
          ) : null}
          {preview.sizing && preview.sizing.stopAttached === false ? (
            <Text variant="caption" color="secondary">
              {t('preview.stopNotAttached')}
            </Text>
          ) : null}
          {(preview.routeLegs ?? []).map((leg) => (
            <Text
              key={`${leg.exchange}-${leg.symbol}`}
              variant="caption"
              color="secondary"
              data-testid="route-leg"
            >
              {t('preview.routeLeg', {
                exchange: leg.exchange,
                qty: formatPrice(leg.quantity),
                price: formatPrice(leg.fillPrice),
                fee: formatPrice(leg.fee),
              })}
            </Text>
          ))}
          {preview.routed && (preview.compare?.savedCash ?? 0) > 0.005 ? (
            <Text variant="caption" color="secondary" data-testid="route-saved">
              {t(preview.side === 'sell' ? 'preview.savedSell' : 'preview.savedBuy', {
                exchange: preview.compare?.exchange,
                amount: formatPrice(preview.compare?.savedCash),
              })}
            </Text>
          ) : null}
          {(preview.missingVenues ?? []).length > 0 ? (
            <Text variant="caption" color="secondary">
              {t('preview.missingVenues', { names: (preview.missingVenues ?? []).join(', ') })}
            </Text>
          ) : null}
          {preview.routed && (preview.maxSlippagePct ?? 0) > 0 ? (
            <Text variant="caption" color="secondary" data-testid="route-slippage">
              {t(preview.side === 'sell' ? 'preview.slippageCapSell' : 'preview.slippageCapBuy', {
                pct: formatPreviewPct(preview.maxSlippagePct),
                price: formatPrice(preview.slippageLimitPrice),
              })}
            </Text>
          ) : null}
          {(preview.routeLegs ?? []).some((leg) => leg.aliased) ? (
            <Text variant="caption" color="secondary">
              {t('preview.aliasQuote')}
            </Text>
          ) : null}
          {(preview.legs ?? []).map((leg, index) => (
            <Text key={`${leg.role ?? 'leg'}-${index}`} variant="caption" color="secondary">
              {leg.role === 'take_profit' ? t('preview.legTakeProfit') : t('preview.legStopLoss')}
              {leg.estimatedRealizedPnl != null
                ? ` · ${t('preview.pnl')} ${formatPrice(leg.estimatedRealizedPnl)}`
                : ''}
            </Text>
          ))}
          {warnings.map((id) => (
            <Alert key={id} type={warningTone(id)} showIcon message={t(`preview.warnings.${id}`)} />
          ))}
          <Text variant="caption" color="secondary">
            {t('preview.note')}
          </Text>
        </>
      ) : null}
    </Panel>
  );
}
