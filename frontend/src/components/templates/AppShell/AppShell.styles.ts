import { Link, NavLink as RouterNavLink } from 'react-router-dom';
import styled from 'styled-components';

export const AppLayout = styled.div`
  display: grid;
  grid-template-columns: 176px minmax(0, 1fr);
  grid-template-rows: minmax(0, 1fr);
  height: 100%;
  min-height: 100%;
  min-width: 0;
  background: ${({ theme }) => theme.semantic.bg.canvas};

  ${({ theme }) => theme.media.tablet} {
    grid-template-columns: minmax(0, 1fr);
    grid-template-rows: auto minmax(0, 1fr);
  }
`;

export const AppSidebar = styled.aside`
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  border-right: 1px solid ${({ theme }) => theme.semantic.border.subtle};

  ${({ theme }) => theme.media.tablet} {
    flex-direction: row;
    align-items: center;
    gap: ${({ theme }) => theme.spacing[2]}px;
    padding: 0 ${({ theme }) => theme.spacing[2]}px;
    border-right: 0;
    border-bottom: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  }
`;

export const AppStage = styled.div`
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  height: 100%;
`;

export const AppTopbar = styled.header`
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: ${({ theme }) => theme.spacing[2]}px;
  min-height: 44px;
  padding: 0 ${({ theme }) => theme.spacing[3]}px;
  background: ${({ theme }) => theme.semantic.bg.page};
  border-bottom: 1px solid ${({ theme }) => theme.semantic.border.subtle};
`;

export const AppWorkspace = styled.main<{ $wide?: boolean; $flush?: boolean }>`
  flex: 1;
  min-width: 0;
  min-height: 0;
  overflow: ${({ $flush }) => ($flush ? 'hidden' : 'auto')};
  display: flex;
  flex-direction: column;
  padding: ${({ theme, $flush }) => ($flush ? 0 : `${theme.spacing[3]}px ${theme.spacing[4]}px`)};
`;

export const AppStatus = styled.footer`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-height: 26px;
  padding: 0 ${({ theme }) => theme.spacing[3]}px;
  border-top: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  background: ${({ theme }) => theme.semantic.bg.canvas};
  color: ${({ theme }) => theme.semantic.text.tertiary};
  font-size: 11px;
`;

/** @deprecated Names kept so existing imports compile. */
export const AppHeader = AppTopbar;
export const AppContent = AppWorkspace;
export const AppFooter = AppStatus;
export const AppBanner = styled.div`
  width: 100%;
  flex-shrink: 0;
`;

export const HeaderBar = styled.div`
  display: contents;
`;

export const HeaderBrand = styled.div`
  padding: ${({ theme }) => theme.spacing[3]}px ${({ theme }) => theme.spacing[3]}px
    ${({ theme }) => theme.spacing[2]}px;

  ${({ theme }) => theme.media.tablet} {
    padding: ${({ theme }) => theme.spacing[1]}px ${({ theme }) => theme.spacing[1]}px
      ${({ theme }) => theme.spacing[1]}px 0;
    flex-shrink: 0;
  }
`;

export const HeaderNav = styled.nav`
  display: flex;
  flex-direction: column;
  gap: 1px;
  padding: ${({ theme }) => theme.spacing[1]}px 0;
  min-width: 0;
  overflow: auto;

  ${({ theme }) => theme.media.tablet} {
    flex-direction: row;
    flex: 1;
    overflow-x: auto;
    scrollbar-width: none;

    &::-webkit-scrollbar {
      display: none;
    }
  }
`;

export const HeaderTools = styled.div`
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: ${({ theme }) => theme.spacing[2]}px;
  min-width: 0;
  width: 100%;

  ${({ theme }) => theme.media.phone} {
    gap: ${({ theme }) => theme.spacing[1]}px;

    .desk-jump-search {
      width: min(148px, 42vw) !important;
      min-width: 0;
    }

    .desk-lang {
      min-width: 68px !important;
      width: 72px;
    }
  }
`;

export const HeaderSpacer = styled.div`
  display: none;
`;

export const BrandLink = styled(Link)`
  display: inline-flex;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[2]}px;
  text-decoration: none;
  color: ${({ theme }) => theme.semantic.text.primary};
  min-width: 0;

  &:hover {
    color: ${({ theme }) => theme.semantic.accent.default};
  }
`;

export const BrandCopy = styled.span`
  display: flex;
  flex-direction: column;
  min-width: 0;

  ${({ theme }) => theme.media.phone} {
    display: none;
  }
`;

export const BrandName = styled.span`
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.16em;
  line-height: 1.2;
  text-transform: uppercase;
`;

export const BrandTagline = styled.span`
  font-size: 10px;
  font-weight: 500;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: ${({ theme }) => theme.semantic.text.tertiary};
`;

export const NavLink = styled(RouterNavLink)`
  display: flex;
  align-items: center;
  gap: 10px;
  text-decoration: none;
  color: ${({ theme }) => theme.semantic.text.secondary};
  font-size: 12px;
  font-weight: 500;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  padding: 9px 16px;
  border-left: 2px solid transparent;
  white-space: nowrap;

  &:hover {
    color: ${({ theme }) => theme.semantic.text.primary};
    background: ${({ theme }) => theme.semantic.bg.hover};
  }

  &.active {
    color: ${({ theme }) => theme.semantic.text.primary};
    background: ${({ theme }) => theme.semantic.bg.hover};
    border-left-color: ${({ theme }) => theme.semantic.accent.default};
  }

  ${({ theme }) => theme.media.tablet} {
    border-left: 0;
    border-bottom: 2px solid transparent;
    padding: 10px 10px 8px;

    &.active {
      border-left-color: transparent;
      border-bottom-color: ${({ theme }) => theme.semantic.accent.default};
      background: transparent;
    }
  }
`;
