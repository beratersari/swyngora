import { useTranslation } from 'react-i18next';
import { TickerTape } from '@/components/molecules/TickerTape';
import { DESK_TAPE_SOURCE_LABEL_KEY } from './DeskPriceTape.constants';
import { Bar, Empty, SourceBtn, SourceGroup, TapeSlot } from './DeskPriceTape.styles';
import { DESK_TAPE_SOURCES, type DeskPriceTapeProps } from './DeskPriceTape.types';

/** Sticky last-price strip with a venue / watchlist source switch. */
export function DeskPriceTape({
  source,
  onSourceChange,
  items,
  isLoading = false,
  emptyLabel,
  sourceAriaLabel,
  tapeAriaLabel,
  paused = false,
  children,
}: DeskPriceTapeProps) {
  const { t } = useTranslation('common');

  return (
    <Bar role="region" aria-label={tapeAriaLabel}>
      <SourceGroup role="tablist" aria-label={sourceAriaLabel}>
        {DESK_TAPE_SOURCES.map((id) => (
          <SourceBtn
            key={id}
            type="button"
            role="tab"
            aria-selected={source === id}
            $active={source === id}
            onClick={() => onSourceChange(id)}
          >
            {t(DESK_TAPE_SOURCE_LABEL_KEY[id])}
          </SourceBtn>
        ))}
      </SourceGroup>
      <TapeSlot>
        {children ? (
          children
        ) : items.length > 0 ? (
          <TickerTape items={items} ariaLabel={tapeAriaLabel} paused={paused} />
        ) : (
          <Empty>
            {isLoading
              ? t('status.loading')
              : (emptyLabel ?? t('tape.empty', { defaultValue: 'No prices yet' }))}
          </Empty>
        )}
      </TapeSlot>
    </Bar>
  );
}
