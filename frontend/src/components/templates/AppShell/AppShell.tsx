import {
  AppBanner,
  AppLayout,
  AppSidebar,
  AppStage,
  AppStatus,
  AppTopbar,
  AppWorkspace,
  HeaderBrand,
  HeaderNav,
  HeaderTools,
} from './AppShell.styles';
import type { AppShellProps } from './AppShell.types';

/**
 * Terminal chrome: venue rail + utility bar + workspace.
 */
export function AppShell({
  brand,
  nav,
  tools,
  footer,
  children,
  navAriaLabel,
  wide = false,
  flush = false,
  banner,
}: AppShellProps) {
  return (
    <AppLayout>
      <AppSidebar>
        <HeaderBrand>{brand}</HeaderBrand>
        <HeaderNav aria-label={navAriaLabel}>{nav}</HeaderNav>
      </AppSidebar>
      <AppStage>
        <AppTopbar>{tools ? <HeaderTools>{tools}</HeaderTools> : null}</AppTopbar>
        {banner ? <AppBanner>{banner}</AppBanner> : null}
        <AppWorkspace $wide={wide} $flush={flush}>
          {children}
        </AppWorkspace>
        {footer ? <AppStatus>{footer}</AppStatus> : null}
      </AppStage>
    </AppLayout>
  );
}
