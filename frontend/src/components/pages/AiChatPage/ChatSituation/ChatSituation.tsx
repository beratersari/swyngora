import { useTranslation } from 'react-i18next';
import { Text } from '@/components/atoms/Text';
import { CoinSituation } from '@/components/molecules/CoinSituation';
import { PositionSituation } from '@/components/molecules/PositionSituation';
import { Read, Stack } from './ChatSituation.styles';
import type { AiSituationEntry, ChatSituationListProps } from './ChatSituation.types';
import { situationEntryCaption } from './helpers';

export function ChatSituationList({ reads }: ChatSituationListProps) {
  return (
    <Stack>
      {reads.map((read, index) => {
        const entries = (read.entries ?? []).filter((entry) => entry.entrySituation);
        return (
          <ChatSituation
            key={`${read.symbol}-${index}`}
            symbol={read.symbol}
            situation={read.situation}
            entries={entries}
          />
        );
      })}
    </Stack>
  );
}

function ChatSituation({
  symbol,
  situation,
  entries,
}: {
  symbol: string;
  situation: AiSituationEntry['entrySituation'];
  entries: AiSituationEntry[];
}) {
  const { t, i18n } = useTranslation(['ai', 'detail']);
  if (entries.length === 1) {
    return (
      <Read aria-label={t('ai:situationSymbol', { symbol })}>
        <Text variant="label">{t('ai:situationSymbol', { symbol })}</Text>
        <EntryCaption entry={entries[0]} locale={i18n.language} />
        <PositionSituation entry={entries[0].entrySituation} now={situation} />
      </Read>
    );
  }
  if (entries.length > 1) {
    return (
      <Read aria-label={t('ai:situationSymbol', { symbol })}>
        <Text variant="label">{t('ai:situationSymbol', { symbol })}</Text>
        {entries.map((entry, index) => (
          <Read key={`${entry.exchange}-${entry.entryAt ?? index}`}>
            <EntryCaption entry={entry} locale={i18n.language} />
            <PositionSituation entry={entry.entrySituation} />
          </Read>
        ))}
        <Text variant="caption" color="secondary">
          {t('detail:situation.now')}
        </Text>
        <CoinSituation report={situation} />
      </Read>
    );
  }
  return (
    <Read aria-label={t('ai:situationSymbol', { symbol })}>
      <Text variant="label">{t('ai:situationSymbol', { symbol })}</Text>
      <CoinSituation report={situation} />
    </Read>
  );
}

function EntryCaption({ entry, locale }: { entry: AiSituationEntry; locale: string }) {
  const { t } = useTranslation('ai');
  const caption = situationEntryCaption(entry, t, locale);
  if (!caption) return null;
  return (
    <Text variant="caption" color="secondary">
      {caption}
    </Text>
  );
}
