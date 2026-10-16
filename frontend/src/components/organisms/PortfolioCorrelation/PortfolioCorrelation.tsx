import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { useGetPortfolioCorrelationQuery } from '@/libs/api';
import { correlationKind, formatCorr, formatSwing } from './helpers';
import type { PortfolioCorrelationProps } from './PortfolioCorrelation.types';

/**
 * How the open coins moved together, next to the risk limits.
 * The ticket shows what one new order would do to the same swing.
 */
export function PortfolioCorrelation({ portfolioId, disabled }: PortfolioCorrelationProps) {
  const { t } = useTranslation('portfolio');
  const query = useGetPortfolioCorrelationQuery(portfolioId ? { portfolioId } : undefined, {
    skip: disabled,
  });
  const report = query.data;
  const kind = correlationKind(report);
  const asset = report?.holdings?.[0]?.asset ?? '';

  return (
    <section data-testid="portfolio-correlation">
      <Text variant="h4" color="primary">
        {t('correlation.title')}
      </Text>
      <Text variant="caption" color="secondary">
        {t('correlation.hint')}
      </Text>
      {query.isLoading ? (
        <Text variant="caption" color="secondary">
          {t('correlation.loading')}
        </Text>
      ) : null}
      {query.isError ? (
        <Text variant="caption" color="secondary">
          {t('correlation.failed')}
        </Text>
      ) : null}
      {report && kind === 'empty' ? (
        <Text variant="bodySm">{t('correlation.empty')}</Text>
      ) : null}
      {report && kind === 'incomplete' ? (
        <Text variant="bodySm">{t('correlation.incomplete')}</Text>
      ) : null}
      {report && kind === 'one' ? (
        <Text variant="bodySm">
          {t('correlation.one', { asset, swing: formatSwing(report.swingPct) })}
        </Text>
      ) : null}
      {report && (kind === 'together' || kind === 'hedge' || kind === 'inverse' || kind === 'mixed') ? (
        <Text variant="bodySm">
          {t(`correlation.${kind}`, {
            corr: formatCorr(report.together),
            swing: formatSwing(report.swingPct),
            apart: formatSwing(report.apartPct),
          })}
        </Text>
      ) : null}
      {(report?.pairs ?? []).map((pair) =>
        pair.complete && pair.corr != null ? (
          <Text key={`${pair.left}-${pair.right}`} variant="caption" color="secondary">
            {t('correlation.pair', {
              left: pair.left,
              right: pair.right,
              corr: formatCorr(pair.corr),
              relation: t(`correlation.relation.${pair.relation ?? 'mixed'}`),
            })}
          </Text>
        ) : null,
      )}
      {(report?.missing ?? []).length > 0 ? (
        <Text variant="caption" color="secondary">
          {t('correlation.missing', { names: report?.missing?.join(', ') })}
        </Text>
      ) : null}
    </section>
  );
}
