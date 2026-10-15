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
