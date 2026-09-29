import { Alert } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { unusualSentence, unusualVenueName } from './helpers';
import { Panel, TitleRow, VenueBlock } from './UnusualContext.styles';
import type { UnusualContextProps, UnusualTranslate } from './UnusualContext.types';

export function UnusualContext({ report, isLoading, error }: UnusualContextProps) {
  const { t } = useTranslation('detail');
  // i18next's key union rejects a plain string key, so the sentence helpers use a small wrapper.
  const translate = t as unknown as UnusualTranslate;
  if (!report && !isLoading && !error) return null;

  return (
    <Panel data-testid="unusual-context">
      <TitleRow>
        <Text variant="h4" color="primary">
          {t('tape.unusual.title')}
        </Text>
        <Text variant="caption" color="secondary">
          {t('tape.unusual.subtitle')}
        </Text>
      </TitleRow>
      {error ? <Alert type="info" showIcon message={error} /> : null}
      {isLoading && !report ? (
        <Text variant="caption" color="secondary">
          {t('tape.unusual.checking')}
        </Text>
      ) : null}
      {report?.venues?.map((venue) => (
        <VenueBlock key={venue.exchange ?? 'venue'}>
          <Text variant="label" color="primary">
            {unusualVenueName(venue.exchange, translate)}
          </Text>
          {venue.readings?.map((reading) => (
            <Text
              key={reading.metric}
              variant="body"
              color="primary"
              data-testid={`unusual-${venue.exchange}-${reading.metric}`}
            >
              {unusualSentence(reading, translate)}
            </Text>
          ))}
        </VenueBlock>
      ))}
    </Panel>
  );
}
