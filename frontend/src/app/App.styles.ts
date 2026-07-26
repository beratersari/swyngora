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

export const HeaderNav = styled.div`
  display: flex;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[4]}px;
`;

export const AppContent = styled(Layout.Content)`
  padding: ${({ theme }) => theme.spacing[6]}px;
  max-width: 1280px;
  width: 100%;
  margin: 0 auto;
`;

export const AppFooter = styled(Layout.Footer)`
  text-align: center;
  background: ${({ theme }) => theme.semantic.bg.canvas};
`;
