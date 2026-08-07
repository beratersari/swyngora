import {
  AppBanner,
  AppContent,
  AppFooter,
  AppHeader,
  AppLayout,
  HeaderNav,
  HeaderSpacer,
} from './AppShell.styles';
import type { AppShellProps } from './AppShell.types';

/**
 * Product chrome: header nav + main content + optional footer.
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
        <HeaderNav aria-label={navAriaLabel}>
          {brand}
          {nav}
        </HeaderNav>
        <HeaderSpacer />
        {tools}
      </AppHeader>
      {banner ? <AppBanner>{banner}</AppBanner> : null}
      <AppContent $wide={wide}>{children}</AppContent>
      {footer ? <AppFooter>{footer}</AppFooter> : null}
    </AppLayout>
  );
}
