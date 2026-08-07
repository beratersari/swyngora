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
      <rect x="1" y="1" width="30" height="30" rx="8" fill="currentColor" opacity="0.12" />
      <path
        d="M8 21.5 14.2 9h3.7L24 21.5h-3.3l-1.1-2.5h-7.2l-1.1 2.5H8Zm5.4-5h5.1l-2.5-5.7-2.6 5.7Z"
        fill="currentColor"
      />
      <path d="M7 23.5h18" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" opacity="0.7" />
    </Mark>
  );
}
