import styled from 'styled-components';
import { Alert } from 'antd';

export const PageStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[4]}px;
  width: 100%;
`;

export const PageIntro = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const MetaRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[3]}px;
  margin-bottom: ${({ theme }) => theme.spacing[2]}px;
`;

export const McapHintAlert = styled(Alert)`
  margin-bottom: ${({ theme }) => theme.spacing[2]}px;
`;
