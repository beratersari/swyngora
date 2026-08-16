import {
  AppBanner,
  AppLayout,
  AppStatus,
  AppTopbar,
  AppWorkspace,
  HeaderBrand,
  HeaderNav,
  HeaderTools,
  StickyChrome,
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
  tape,
}: AppShellProps) {
  return (
    <AppLayout>
      <StickyChrome>
        {tape}
        <AppTopbar>
          <HeaderBrand>{brand}</HeaderBrand>
          <HeaderNav aria-label={navAriaLabel}>{nav}</HeaderNav>
          {tools ? <HeaderTools>{tools}</HeaderTools> : null}
        </AppTopbar>
      </StickyChrome>
      {banner ? <AppBanner>{banner}</AppBanner> : null}
      <AppWorkspace $wide={wide} $flush={flush}>
        {children}
      </AppWorkspace>
      {footer ? <AppStatus>{footer}</AppStatus> : null}
    </AppLayout>
  );
}
