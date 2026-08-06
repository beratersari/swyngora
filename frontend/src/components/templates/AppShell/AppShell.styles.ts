import { Link, NavLink as RouterNavLink } from 'react-router-dom';
import styled from 'styled-components';
import { Layout } from 'antd';

export const AppLayout = styled(Layout)`
  min-height: 100%;
  background: ${({ theme }) => theme.semantic.bg.canvas};
`;

export const AppHeader = styled(Layout.Header)`
  display: flex;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[4]}px;
  padding-inline: ${({ theme }) => theme.spacing[6]}px;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  border-bottom: 1px solid ${({ theme }) => theme.semantic.border.default};
  line-height: 1;
`;

export const HeaderNav = styled.nav`
  display: flex;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[4]}px;
`;

export const BrandLink = styled(Link)`
  text-decoration: none;
`;

/** React Router NavLink — `.active` + `aria-current="page"` for wayfinding. */
export const NavLink = styled(RouterNavLink)`
  text-decoration: none;
  color: ${({ theme }) => theme.semantic.text.secondary};
  font-size: ${({ theme }) => theme.typeScale.label.fontSize}px;
  font-weight: ${({ theme }) => theme.typeScale.label.fontWeight};
  line-height: ${({ theme }) => theme.typeScale.label.lineHeight};
  letter-spacing: ${({ theme }) => theme.typeScale.label.letterSpacing};
  padding-block: ${({ theme }) => theme.spacing[1]}px;
  border-bottom: 2px solid transparent;

  &:hover {
    color: ${({ theme }) => theme.semantic.text.primary};
  }

  &.active {
    color: ${({ theme }) => theme.palette.caribbeanGreen};
    border-bottom-color: ${({ theme }) => theme.palette.caribbeanGreen};
  }
`;
export const HeaderSpacer = styled.div`
  flex: 1;
`;

export const AppContent = styled(Layout.Content)<{ $wide?: boolean }>`
  padding: ${({ theme }) => theme.spacing[6]}px;
  max-width: ${({ $wide }) => ($wide ? '1600px' : '1280px')};
  width: 100%;
  margin: 0 auto;
`;

export const AppBanner = styled.div`
  padding: ${({ theme }) => theme.spacing[2]}px ${({ theme }) => theme.spacing[6]}px;
  max-width: 1600px;
  width: 100%;
  margin: 0 auto;
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[2]}px;
  align-items: center;
`;

export const AppFooter = styled(Layout.Footer)`
  text-align: center;
  background: ${({ theme }) => theme.semantic.bg.canvas};
`;
