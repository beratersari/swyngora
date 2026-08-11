import {
  AppBanner,
  AppContent,
  AppFooter,
  AppHeader,
  AppLayout,
  HeaderBar,
  HeaderBrand,
  HeaderNav,
  HeaderTools,
} from './AppShell.styles';
import type { AppShellProps } from './AppShell.types';

/**
 * Product chrome: command bar (brand · nav · tools), optional tape, footer.
 * On tablet/phone the bar wraps: brand+tools on row 1, scrollable nav on row 2.
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
        <HeaderBar>
          <HeaderBrand>{brand}</HeaderBrand>
          <HeaderNav aria-label={navAriaLabel}>{nav}</HeaderNav>
          {tools ? <HeaderTools>{tools}</HeaderTools> : null}
        </HeaderBar>
      </AppHeader>
      {banner ? <AppBanner>{banner}</AppBanner> : null}
      <AppContent $wide={wide}>{children}</AppContent>
      {footer ? <AppFooter>{footer}</AppFooter> : null}
    </AppLayout>
  );
}
