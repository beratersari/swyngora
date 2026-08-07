import { useEffect, useRef, useState, type ReactNode } from 'react';
import { numericFlashDirection } from './FlashValue.helpers';
import { FlashWrap } from './FlashValue.styles';
import type { FlashValueProps } from './FlashValue.types';

const FLASH_MS = 650;

/** Brief green/red wash when a live numeric tick changes. */
export function FlashValue({ value, children, className }: FlashValueProps) {
  const prev = useRef<unknown>(value);
  const [dir, setDir] = useState<'up' | 'down' | null>(null);

  useEffect(() => {
    const nextDir = numericFlashDirection(prev.current, value);
    prev.current = value;
    if (!nextDir) return undefined;
    setDir(nextDir);
    const id = window.setTimeout(() => setDir(null), FLASH_MS);
    return () => window.clearTimeout(id);
  }, [value]);

  return (
    <FlashWrap $dir={dir} className={className} data-flash={dir ?? undefined}>
      {children as ReactNode}
    </FlashWrap>
  );
}
