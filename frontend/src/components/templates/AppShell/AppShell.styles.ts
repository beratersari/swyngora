import { Link, NavLink as RouterNavLink } from 'react-router-dom';
import styled from 'styled-components';
import { Layout } from 'antd';

export const AppLayout = styled(Layout)`
  min-height: 100%;
  min-width: 0;
  background: ${({ theme }) => theme.semantic.bg.canvas};
`;

export const AppHeader = styled(Layout.Header)`
  display: flex;
  flex-direction: column;
  align-items: stretch;
  height: auto;
  line-height: 1.2;
  padding: 0;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  border-bottom: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  border-top: 2px solid ${({ theme }) => theme.semantic.accent.default};
  position: sticky;
  top: 0;
  z-index: 40;
`;

export const HeaderBar = styled.div`
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-height: 48px;
  padding: 0 ${({ theme }) => theme.spacing[5]}px;

  ${({ theme }) => theme.media.tablet} {
    grid-template-columns: auto minmax(0, 1fr);
    grid-template-areas:
      'brand tools'
      'nav nav';
    gap: ${({ theme }) => theme.spacing[2]}px ${({ theme }) => theme.spacing[3]}px;
    padding: ${({ theme }) => theme.spacing[2]}px ${({ theme }) => theme.spacing[3]}px
      ${({ theme }) => theme.spacing[2]}px;
  }

  ${({ theme }) => theme.media.phone} {
    padding: ${({ theme }) => theme.spacing[2]}px ${({ theme }) => theme.spacing[2]}px
      ${({ theme }) => theme.spacing[1]}px;
  }
`;

/** @deprecated Use HeaderBar — kept so older snapshots keep compiling. */
export const HeaderTop = HeaderBar;

export const HeaderBrand = styled.div`
  min-width: 0;

  ${({ theme }) => theme.media.tablet} {
    grid-area: brand;
  }
`;

export const HeaderNav = styled.nav`
  display: flex;
  align-items: center;
  gap: 2px;
  min-width: 0;
  overflow-x: auto;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;

  &::-webkit-scrollbar {
    display: none;
  }

  ${({ theme }) => theme.media.tablet} {
    grid-area: nav;
    padding-bottom: 2px;
  }
`;

export const BrandLink = styled(Link)`
  display: inline-flex;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[2]}px;
  text-decoration: none;
  color: ${({ theme }) => theme.semantic.text.primary};
  min-width: 0;
  transition: color ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard};

  &:hover {
    color: ${({ theme }) => theme.semantic.accent.default};
  }
`;

export const BrandCopy = styled.span`
  display: flex;
  flex-direction: column;
  min-width: 0;

  ${({ theme }) => theme.media.xs} {
    display: none;
  }
`;

export const BrandName = styled.span`
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.14em;
  line-height: 1.15;
  text-transform: uppercase;
`;

export const BrandTagline = styled.span`
  font-size: 10px;
  font-weight: 500;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: ${({ theme }) => theme.semantic.text.tertiary};
`;

export const NavLink = styled(RouterNavLink)`
  text-decoration: none;
  color: ${({ theme }) => theme.semantic.text.secondary};
  font-size: 12px;
  font-weight: 500;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  padding: 14px 10px 12px;
  border-radius: 0;
  white-space: nowrap;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  transition:
    color ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard},
    border-color ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard};

  &:hover {
    color: ${({ theme }) => theme.semantic.text.primary};
    background: transparent;
  }

  &.active {
    color: ${({ theme }) => theme.semantic.text.primary};
    background: transparent;
    border-bottom-color: ${({ theme }) => theme.semantic.accent.default};
    box-shadow: none;
  }
`;

export const HeaderSpacer = styled.div`
  display: none;
`;

export const HeaderTools = styled.div`
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: ${({ theme }) => theme.spacing[2]}px;
  min-width: 0;

  ${({ theme }) => theme.media.tablet} {
    grid-area: tools;
  }

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

export const AppContent = styled(Layout.Content)<{ $wide?: boolean }>`
  padding: ${({ theme }) => theme.spacing[4]}px ${({ theme }) => theme.spacing[5]}px
    ${({ theme }) => theme.spacing[8]}px;
  max-width: ${({ $wide }) => ($wide ? '1600px' : '1200px')};
  width: 100%;
  min-width: 0;
  margin: 0 auto;

  ${({ theme }) => theme.media.tablet} {
    padding: ${({ theme }) => theme.spacing[3]}px ${({ theme }) => theme.spacing[3]}px
      ${({ theme }) => theme.spacing[5]}px;
  }

  ${({ theme }) => theme.media.phone} {
    padding: ${({ theme }) => theme.spacing[3]}px ${({ theme }) => theme.spacing[2]}px
      ${({ theme }) => theme.spacing[4]}px;
  }
`;

export const AppBanner = styled.div`
  width: 100%;
`;

export const AppFooter = styled(Layout.Footer)`
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  text-align: left;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  border-top: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  padding: ${({ theme }) => theme.spacing[3]}px ${({ theme }) => theme.spacing[5]}px
    ${({ theme }) => theme.spacing[4]}px;
`;
