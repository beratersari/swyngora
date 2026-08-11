import { Link, NavLink as RouterNavLink } from 'react-router-dom';
import styled from 'styled-components';
import { Layout } from 'antd';

export const AppLayout = styled(Layout)`
  min-height: 100%;
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
  border-bottom: 1px solid ${({ theme }) => theme.semantic.border.default};
  position: sticky;
  top: 0;
  z-index: 40;
`;

export const HeaderTop = styled.div`
  display: flex;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[3]}px;
  padding: ${({ theme }) => theme.spacing[2]}px ${({ theme }) => theme.spacing[6]}px;
`;

export const HeaderNav = styled.nav`
  display: flex;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[1]}px;
  padding: 0 ${({ theme }) => theme.spacing[6]}px ${({ theme }) => theme.spacing[2]}px;
  overflow-x: auto;
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
`;

export const BrandName = styled.span`
  font-size: 16px;
  font-weight: 700;
  letter-spacing: -0.03em;
  line-height: 1.15;
`;

export const BrandTagline = styled.span`
  font-size: 11px;
  font-weight: 500;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: ${({ theme }) => theme.semantic.text.secondary};
`;

export const NavLink = styled(RouterNavLink)`
  text-decoration: none;
  color: ${({ theme }) => theme.semantic.text.secondary};
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.01em;
  padding: 6px 12px;
  border-radius: ${({ theme }) => theme.radii.pill}px;
  white-space: nowrap;
  transition:
    color ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard},
    background ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard},
    box-shadow ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard},
    transform ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard};

  &:hover {
    color: ${({ theme }) => theme.semantic.text.primary};
    background: ${({ theme }) => theme.semantic.bg.hover};
    transform: translateY(-1px);
  }

  &.active {
    color: ${({ theme }) => theme.semantic.text.primary};
    background: ${({ theme }) => theme.semantic.bg.accentSoft};
    box-shadow: inset 0 0 0 1px ${({ theme }) => theme.semantic.border.accent};
    transform: none;
  }

  @media (prefers-reduced-motion: reduce) {
    &:hover {
      transform: none;
    }
  }
`;

export const HeaderSpacer = styled.div`
  flex: 1;
`;

export const HeaderTools = styled.div`
  display: flex;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[2]}px;
  min-width: 0;
`;

export const AppContent = styled(Layout.Content)<{ $wide?: boolean }>`
  padding: ${({ theme }) => theme.spacing[5]}px ${({ theme }) => theme.spacing[6]}px
    ${({ theme }) => theme.spacing[8]}px;
  max-width: ${({ $wide }) => ($wide ? '1600px' : '1280px')};
  width: 100%;
  margin: 0 auto;
`;

export const AppBanner = styled.div`
  width: 100%;
`;

export const AppFooter = styled(Layout.Footer)`
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[1]}px;
  text-align: center;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  border-top: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  padding: ${({ theme }) => theme.spacing[4]}px ${({ theme }) => theme.spacing[6]}px;
`;
