import { Text } from '@/components/atoms/Text';
import { Copy, Extra, Eyebrow, Wrap } from './PageHeader.styles';
import type { PageHeaderProps } from './PageHeader.types';

/** Consistent desk page title block (eyebrow + h1 + subtitle + actions). */
export function PageHeader({ title, subtitle, eyebrow, extra }: PageHeaderProps) {
  return (
    <Wrap>
      <Copy>
        {eyebrow ? <Eyebrow>{eyebrow}</Eyebrow> : null}
        <Text variant="h2" color="primary" as="h1">
          {title}
        </Text>
        {subtitle ? (
          <Text variant="body" color="secondary">
            {subtitle}
          </Text>
        ) : null}
      </Copy>
      {extra ? <Extra>{extra}</Extra> : null}
    </Wrap>
  );
}
