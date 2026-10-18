import { formatPrice } from '@/libs/utils/formatPrice';

/** A positive difference gets a plus so paid-more is visible next to the quote. */
export function signedQuote(value: number | undefined): string {
  const text = formatPrice(value);
  if (value != null && value > 0 && !text.startsWith('+')) return `+${text}`;
  return text;
}
