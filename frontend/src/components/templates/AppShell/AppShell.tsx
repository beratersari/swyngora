import {
  AppBanner,
  AppContent,
  AppFooter,
  AppHeader,
  AppLayout,
  HeaderNav,
  HeaderSpacer,
  HeaderTools,
  HeaderTop,
} from './AppShell.styles';
import type { AppShellProps } from './AppShell.types';

/**
 * Product chrome: sticky brand/tools row, nav pills, optional ticker banner, footer.
 * Data-free layout shell (Atomic Design template).
 */
export function AppShell({
  brand,
  nav,
  tools,
  footer,
  children,
  navAriaLabel,
  wide = false,
  banner,
}: AppShellProps) {
  return (
    <AppLayout>
      <AppHeader>
        <HeaderTop>
          {brand}
          <HeaderSpacer />
          {tools ? <HeaderTools>{tools}</HeaderTools> : null}
        </HeaderTop>
        <HeaderNav aria-label={navAriaLabel}>{nav}</HeaderNav>
      </AppHeader>
      {banner ? <AppBanner>{banner}</AppBanner> : null}
      <AppContent $wide={wide}>{children}</AppContent>
      {footer ? <AppFooter>{footer}</AppFooter> : null}
    </AppLayout>
  );
}
