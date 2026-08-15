import { Mark } from './BrandMark.styles';
import type { BrandMarkProps } from './BrandMark.types';

/** Compact wordmark glyph for app chrome. */
export function BrandMark({ size = 28, title }: BrandMarkProps) {
  return (
    <Mark
      width={size}
      height={size}
      viewBox="0 0 32 32"
      aria-hidden={title ? undefined : true}
      role={title ? 'img' : undefined}
    >
      {title ? <title>{title}</title> : null}
      <rect x="2" y="2" width="28" height="28" rx="2" fill="currentColor" opacity="0.1" />
      <path d="M8 22V14h3v8H8Zm6.5 0V8h3v14h-3ZM21 22v-6h3v6h-3Z" fill="currentColor" />
    </Mark>
  );
}
