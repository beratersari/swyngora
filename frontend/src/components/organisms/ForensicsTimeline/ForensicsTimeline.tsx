import { Skeleton } from 'antd';
import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { formatForensicsTime } from '@/libs/utils/forensics';
import { Body, Card, Dot, Item, List, Meta, Rail } from './ForensicsTimeline.styles';
import type { ForensicsTimelineProps } from './ForensicsTimeline.types';

export function ForensicsTimeline({ timeline, isLoading }: ForensicsTimelineProps) {
  const { t } = useTranslation('forensics');
  if (isLoading) {
    return <Skeleton active paragraph={{ rows: 6 }} />;
  }
  const events = timeline?.events?.filter((ev) => ev.title) ?? [];
  if (events.length === 0) {
    return null;
  }
  return (
    <Card aria-label={t('timeline.title')}>
      <Text variant="bodySm">{t('timeline.title')}</Text>
      {timeline?.knownUntil ? (
        <Text variant="caption" color="secondary">
          {t('timeline.knownUntil', { time: formatForensicsTime(timeline.knownUntil) })}
        </Text>
      ) : null}
      {timeline?.summary ? (
        <Text variant="bodySm" color="secondary">
          {timeline.summary}
        </Text>
      ) : null}
      <List>
        {events.map((ev, i) => (
          <Item key={`${ev.at ?? i}-${ev.kind ?? ''}-${ev.venue ?? ''}-${ev.title ?? ''}`}>
            <Rail>
              <Dot />
            </Rail>
            <Body>
              <Meta>
                <Text variant="caption" color="secondary">
                  {formatForensicsTime(ev.at)}
                </Text>
                {ev.offsetLabel ? (
                  <Text variant="caption" color="secondary">
                    {ev.offsetLabel}
                  </Text>
                ) : null}
                {ev.orderUnknown ? (
                  <Text variant="caption" color="secondary">
                    {t('timeline.orderUnknown')}
                  </Text>
                ) : null}
              </Meta>
              <Text variant="bodySm">{ev.title}</Text>
              {ev.detail ? (
                <Text variant="caption" color="secondary">
                  {ev.detail}
                </Text>
              ) : null}
            </Body>
          </Item>
        ))}
      </List>
    </Card>
  );
}
