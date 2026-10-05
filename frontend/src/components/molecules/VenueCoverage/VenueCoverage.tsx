import { Alert } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { coverageAge, venueRows, venueStatusKey } from './helpers';
import type { VenueCoverageProps } from './VenueCoverage.types';

export function VenueCoverage({ coverage }: VenueCoverageProps) {
  const { t } = useTranslation('common');
  const venues = venueRows(coverage?.venues);
  if (venues.length === 0) return null;
  const complete = Boolean(coverage?.complete);
  return (
    <Alert
      data-testid="venue-coverage"
      type={complete ? 'success' : 'warning'}
      showIcon
      message={complete ? t('coverage.complete') : t('coverage.incomplete')}
      description={
        <div>
          {venues.map((venue) => {
            const status = venueStatusKey(venue.status);
            const age = status === 'stale' ? coverageAge(venue.ageSec) : '';
            const name = t(`exchanges.${venue.exchange}`, { defaultValue: venue.exchange ?? '' });
            return (
              <Text key={venue.exchange} variant="caption" color="secondary" style={{ display: 'block' }}>
                {name}: {t(`coverage.${status}`)}
                {age ? ` (${t('coverage.age', { age })})` : ''}
              </Text>
            );
          })}
        </div>
      }
    />
  );
}
