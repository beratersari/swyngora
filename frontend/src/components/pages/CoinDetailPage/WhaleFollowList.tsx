import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import type { MarketDeskReport } from '@/libs/api';

type WhaleEvent = NonNullable<MarketDeskReport['events']>[number];

const intentKey: Record<string, string> = {
  open_long: 'flow.whale.openLong',
  close_short: 'flow.whale.closeShort',
  open_short: 'flow.whale.openShort',
  close_long: 'flow.whale.closeLong',
  unclear: 'flow.whale.unclear',
};

export function WhaleFollowList({ events }: { events?: WhaleEvent[] }) {
  const { t } = useTranslation('detail');
  if (!events || events.length === 0) return null;
  return (
    <div data-testid="whale-follow">
      {events.slice(0, 8).map((ev, i) => (
        <Text key={`${ev.symbol}-${ev.side}-${i}`} variant="caption" color="secondary">
          {t('flow.whale.row', {
            side: ev.side,
            notional: ev.notional,
            intent: t(intentKey[ev.intent || ''] || 'flow.whale.unclear'),
            oi: ev.oiChangePct || '—',
            after: ev.afterReady
              ? t('flow.whale.after', { price: ev.priceAfterPct || '—', oi: ev.oiAfterPct || '—' })
              : t('flow.whale.afterPending'),
          })}
        </Text>
      ))}
    </div>
  );
}
