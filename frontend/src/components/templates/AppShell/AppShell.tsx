import {
  AppBanner,
  AppLayout,
  AppStatus,
  AppTopbar,
  AppWorkspace,
  HeaderBrand,
  HeaderNav,
  HeaderTools,
} from './AppShell.styles';
import type { AppShellProps } from './AppShell.types';

/**
 * Consumer market chrome: logo + text nav + search (CoinMarketCap-like).
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
      <AppTopbar>
        <HeaderBrand>{brand}</HeaderBrand>
        <HeaderNav aria-label={navAriaLabel}>{nav}</HeaderNav>
        {tools ? <HeaderTools>{tools}</HeaderTools> : null}
      </AppTopbar>
      {banner ? <AppBanner>{banner}</AppBanner> : null}
      <AppWorkspace $wide={wide} $flush={flush}>
        {children}
      </AppWorkspace>
      {footer ? <AppStatus>{footer}</AppStatus> : null}
    </AppLayout>
  );
}
