import type { TFunction } from 'i18next';
import { formatPrice } from '@/libs/utils/formatPrice';
import type { FillPocketMark, FillPocketPriceLine } from './FillPocket.types';

function gapLabel(pct: number): string {
  const n = Math.round(pct * 10) / 10;
  return Number.isInteger(n) ? String(n) : n.toFixed(1);
}

function venueList(missing: string[] | undefined, t: TFunction<'detail'>): string {
  return (missing ?? [])
    .map((id) => t(`situation.venues.${id}`, { defaultValue: id }))
    .join(', ');
}

function sideLabel(side: string | undefined, t: TFunction<'detail'>): string {
  if (side === 'short') return t('fillPocket.sideShort');
  return t('fillPocket.sideLong');
}

/** Sentences for the chart caption and the Trade tab. An empty list draws nothing. */
export function fillPocketSentences(mark: FillPocketMark | null | undefined, t: TFunction<'detail'>): string[] {
  if (!mark?.open || mark.equity) return [];
  const lines: string[] = [];
  const fill = formatPrice(mark.fillPrice);
  const venues = venueList(mark.missing, t);
  if (mark.fillPrice != null && mark.fillPrice > 0) {
    if (mark.missing && mark.missing.length > 0) {
      lines.push(t('fillPocket.fillMissing', { fill, venues }));
    } else if (mark.pocketSide === 'none' || mark.pocketPrice == null || mark.pocketPrice <= 0) {
      lines.push(t('fillPocket.fillNone', { fill }));
    } else if (mark.gapPct === 0 || !mark.pocketFromFill) {
      lines.push(t('fillPocket.fillSame', { fill, side: sideLabel(mark.pocketSide, t) }));
    } else if (mark.pocketFromFill === 'above') {
      lines.push(
        t('fillPocket.fillAbove', {
          fill,
          side: sideLabel(mark.pocketSide, t),
          pct: gapLabel(mark.gapPct ?? 0),
          pocket: formatPrice(mark.pocketPrice),
        }),
      );
    } else {
      lines.push(
        t('fillPocket.fillBelow', {
          fill,
          side: sideLabel(mark.pocketSide, t),
          pct: gapLabel(mark.gapPct ?? 0),
          pocket: formatPrice(mark.pocketPrice),
        }),
      );
    }
  }
  if (mark.marginLiqPrice != null && mark.marginLiqPrice > 0) {
    const price = formatPrice(mark.marginLiqPrice);
    if (mark.missing && mark.missing.length > 0) {
      lines.push(t('fillPocket.marginUnmarked', { price, venues }));
    } else if (mark.pocketPrice != null && mark.pocketPrice > 0 && mark.marginGapPct === 0) {
      lines.push(t('fillPocket.marginSame', { price }));
    } else if (mark.pocketPrice != null && mark.pocketPrice > 0 && mark.marginFromPocket === 'above') {
      lines.push(t('fillPocket.marginAbove', { price, pct: gapLabel(mark.marginGapPct ?? 0) }));
    } else if (mark.pocketPrice != null && mark.pocketPrice > 0 && mark.marginFromPocket === 'below') {
      lines.push(t('fillPocket.marginBelow', { price, pct: gapLabel(mark.marginGapPct ?? 0) }));
    } else if (!mark.pocketPrice) {
      lines.push(t('fillPocket.marginUnmarked', { price, venues }));
    }
  }
  return lines;
}

/** Horizontal chart marks. A missing pocket adds no pocket line. */
export function fillPocketPriceLines(
  mark: FillPocketMark | null | undefined,
  t: TFunction<'detail'>,
): FillPocketPriceLine[] {
  if (!mark?.open || mark.equity) return [];
  const lines: FillPocketPriceLine[] = [];
  if (mark.fillPrice != null && mark.fillPrice > 0) {
    lines.push({ id: 'paper-fill', price: mark.fillPrice, role: 'fill', title: t('fillPocket.lineFill') });
  }
  if (mark.pocketPrice != null && mark.pocketPrice > 0 && !(mark.missing && mark.missing.length > 0)) {
    const pct = mark.gapPct != null ? gapLabel(mark.gapPct) : '';
    const where =
      mark.pocketFromFill === 'above'
        ? t('fillPocket.lineAbove', { pct })
        : mark.pocketFromFill === 'below'
          ? t('fillPocket.lineBelow', { pct })
          : t('fillPocket.lineSame');
    lines.push({
      id: 'paper-pocket',
      price: mark.pocketPrice,
      role: 'pocket',
      title: `${t(mark.pocketSide === 'short' ? 'fillPocket.linePocketShort' : 'fillPocket.linePocketLong')} · ${where}`,
    });
  }
  if (mark.marginLiqPrice != null && mark.marginLiqPrice > 0) {
    lines.push({
      id: 'paper-liq',
      price: mark.marginLiqPrice,
      role: 'margin',
      title: t('fillPocket.lineMargin'),
    });
  }
  return lines;
}
