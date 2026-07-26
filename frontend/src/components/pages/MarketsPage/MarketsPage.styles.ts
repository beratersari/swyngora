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
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[3]}px;
  padding: ${({ theme }) => theme.spacing[3]}px ${({ theme }) => theme.spacing[4]}px;
  background: ${({ theme }) => theme.colors.pine};
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.md}px;
`;

export const MetaLeft = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

export const MetaRight = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

/** High-visibility match count */
export const ResultsBadge = styled.div`
  display: inline-flex;
  align-items: baseline;
  gap: ${({ theme }) => theme.spacing[2]}px;
  padding: ${({ theme }) => theme.spacing[1]}px ${({ theme }) => theme.spacing[3]}px;
  background: rgba(0, 255, 129, 0.1);
  border: 1px solid ${({ theme }) => theme.colors.caribbeanGreen};
  border-radius: ${({ theme }) => theme.radii.pill}px;
`;

export const ResultsCount = styled.span`
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-size: 16px;
  font-weight: 600;
  color: ${({ theme }) => theme.colors.caribbeanGreen};
  letter-spacing: -0.02em;
`;

export const ResultsLabel = styled.span`
  font-size: 13px;
  font-weight: 500;
  color: ${({ theme }) => theme.colors.pistachio};
`;

export const McapHintAlert = styled(Alert)`
  margin-bottom: ${({ theme }) => theme.spacing[2]}px;
`;
