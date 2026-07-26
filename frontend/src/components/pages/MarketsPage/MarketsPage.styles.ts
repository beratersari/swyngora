import styled from 'styled-components';
import { Card } from 'antd';

export const PageStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[6]}px;
  width: 100%;
`;

export const PageIntro = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const PanelCard = styled(Card)`
  background: ${({ theme }) => theme.semantic.bg.muted};
  border-color: ${({ theme }) => theme.semantic.border.default};

  .ant-card-head {
    border-bottom-color: ${({ theme }) => theme.semantic.border.default};
    color: ${({ theme }) => theme.semantic.text.primary};
  }

  .ant-card-body {
    background: ${({ theme }) => theme.semantic.bg.muted};
  }
`;

export const StatusRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

export const BlockSpacer = styled.div`
  margin-top: ${({ theme }) => theme.spacing[3]}px;
`;
