import { Link, NavLink as RouterNavLink } from 'react-router-dom';
import styled from 'styled-components';
import { Layout } from 'antd';

export const AppLayout = styled(Layout)`
  min-height: 100%;
  background: ${({ theme }) => theme.semantic.bg.page};
`;

export const AppHeader = styled(Layout.Header)`
  display: flex;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[4]}px;
  padding-inline: ${({ theme }) => theme.spacing[6]}px;
  background: ${({ theme }) => theme.semantic.bg.chrome};
  border-bottom: 1px solid ${({ theme }) => theme.semantic.border.default};
  line-height: 1;
`;

export const HeaderNav = styled.nav`
  display: flex;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const BrandLink = styled(Link)`
  text-decoration: none;
  margin-right: ${({ theme }) => theme.spacing[3]}px;
  color: ${({ theme }) => theme.semantic.text.primary};

  &:hover {
    color: ${({ theme }) => theme.semantic.accent.default};
  }
`;

/** Primary nav — active state uses brand accent, not neon chart green */
export const NavLink = styled(RouterNavLink)`
  text-decoration: none;
  color: ${({ theme }) => theme.semantic.text.secondary};
  padding: ${({ theme }) => theme.spacing[2]}px ${({ theme }) => theme.spacing[3]}px;
  border-radius: ${({ theme }) => theme.radii.sm}px;
  transition:
    color 0.15s ease,
    background 0.15s ease;

  &:hover {
    color: ${({ theme }) => theme.semantic.text.primary};
    background: ${({ theme }) => theme.semantic.bg.hover};
  }

  &.active {
    color: ${({ theme }) => theme.semantic.text.primary};
    background: ${({ theme }) => theme.semantic.bg.accentSoft};
    box-shadow: inset 0 -2px 0 ${({ theme }) => theme.semantic.accent.default};
  }

  /* Text atom inside inherits via color: inherit when needed */
  span {
    color: inherit;
  }
`;

export const HeaderSpacer = styled.div`
  flex: 1;
`;

export const AppContent = styled(Layout.Content)`
  padding: ${({ theme }) => theme.spacing[6]}px;
  max-width: 1280px;
  width: 100%;
  margin: 0 auto;
  background: transparent;
`;

export const AppFooter = styled(Layout.Footer)`
  text-align: center;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  border-top: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  color: ${({ theme }) => theme.semantic.text.tertiary};
`;
