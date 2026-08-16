import { Link, NavLink as RouterNavLink } from 'react-router-dom';
import styled from 'styled-components';

export const AppLayout = styled.div`
  display: flex;
  flex-direction: column;
  min-height: 100%;
  min-width: 0;
  background: ${({ theme }) => theme.semantic.bg.page};
`;

export const AppSidebar = styled.aside`
  display: contents;
`;

export const AppStage = styled.div`
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
`;

export const StickyChrome = styled.div`
  position: sticky;
  top: 0;
  z-index: 50;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  box-shadow: 0 8px 24px rgba(13, 20, 33, 0.06);
`;

export const AppTopbar = styled.header`
  position: relative;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px 20px;
  min-height: 64px;
  padding: 10px 24px;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  border-bottom: 1px solid ${({ theme }) => theme.semantic.border.default};
`;

export const AppWorkspace = styled.main<{ $wide?: boolean; $flush?: boolean }>`
  flex: 1;
  min-width: 0;
  width: 100%;
  max-width: ${({ $flush }) => ($flush ? 'none' : '1280px')};
  margin: 0 auto;
  padding: ${({ theme, $flush }) => ($flush ? 0 : `${theme.spacing[5]}px ${theme.spacing[4]}px ${theme.spacing[8]}px`)};
`;

export const AppStatus = styled.footer`
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 16px;
  padding: 20px 24px 28px;
  color: ${({ theme }) => theme.semantic.text.tertiary};
  font-size: 12px;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  border-top: 1px solid ${({ theme }) => theme.semantic.border.default};
`;

export const AppHeader = AppTopbar;
export const AppContent = AppWorkspace;
export const AppFooter = AppStatus;
export const AppBanner = styled.div`
  width: 100%;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  border-bottom: 1px solid ${({ theme }) => theme.semantic.border.default};
`;

export const HeaderBar = styled.div`
  display: contents;
`;

export const HeaderBrand = styled.div`
  flex-shrink: 0;
`;

export const HeaderNav = styled.nav`
  display: flex;
  align-items: center;
  gap: 4px;
  flex: 1;
  min-width: 0;
  overflow-x: auto;
  scrollbar-width: none;

  &::-webkit-scrollbar {
    display: none;
  }
`;

export const HeaderTools = styled.div`
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  margin-left: auto;
  min-width: 0;

  ${({ theme }) => theme.media.phone} {
    width: 100%;

    .desk-jump-search {
      flex: 1;
      width: auto !important;
    }
  }
`;

export const HeaderSpacer = styled.div`
  display: none;
`;

export const BrandLink = styled(Link)`
  display: inline-flex;
  align-items: center;
  gap: 10px;
  text-decoration: none;
  color: ${({ theme }) => theme.semantic.text.primary};

  &:hover {
    color: ${({ theme }) => theme.semantic.accent.default};
  }
`;

export const BrandCopy = styled.span`
  display: flex;
  flex-direction: column;
  min-width: 0;
`;

export const BrandName = styled.span`
  font-size: 18px;
  font-weight: 800;
  letter-spacing: -0.03em;
  line-height: 1.1;
  text-transform: none;
`;

export const BrandTagline = styled.span`
  display: none;
`;

export const NavLink = styled(RouterNavLink)`
  display: inline-flex;
  align-items: center;
  text-decoration: none;
  color: ${({ theme }) => theme.semantic.text.primary};
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 0;
  text-transform: none;
  padding: 8px 10px;
  border: 0;
  border-radius: 8px;
  white-space: nowrap;

  &:hover {
    color: ${({ theme }) => theme.semantic.accent.default};
    background: ${({ theme }) => theme.semantic.bg.hover};
  }

  &.active {
    color: ${({ theme }) => theme.semantic.accent.default};
    background: transparent;
    border: 0;
  }
`;
