import type { BrandTagProps } from './BrandTag.types';
import { StyledBrandTag } from './BrandTag.styles';

/**
 * Brand-colored status / exchange tags — avoids Ant default blue “processing”.
 */
export function BrandTag({ variant = 'status', children, className }: BrandTagProps) {
  return (
    <StyledBrandTag $variant={variant} className={className} bordered={false}>
      {children}
    </StyledBrandTag>
  );
}
