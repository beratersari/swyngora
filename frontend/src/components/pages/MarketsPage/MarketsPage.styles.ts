import styled from 'styled-components';
import { Alert } from 'antd';

export const PageStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  width: 100%;
  min-width: 0;
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
  gap: ${({ theme }) => theme.spacing[2]}px;
  padding: 2px 2px 0;
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

/** Match count — brand accent (mountain meadow), not chart neon */
export const ResultsBadge = styled.div`
  display: inline-flex;
  align-items: baseline;
  gap: ${({ theme }) => theme.spacing[2]}px;
  padding: ${({ theme }) => theme.spacing[1]}px ${({ theme }) => theme.spacing[3]}px;
  background: ${({ theme }) => theme.semantic.bg.accentSoft};
  border: 1px solid ${({ theme }) => theme.semantic.border.accent};
  border-radius: ${({ theme }) => theme.radii.pill}px;
`;

export const ResultsCount = styled.span`
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-size: 16px;
  font-weight: 600;
  color: ${({ theme }) => theme.semantic.accent.default};
  letter-spacing: -0.02em;
`;

export const ResultsLabel = styled.span`
  font-size: 13px;
  font-weight: 500;
  color: ${({ theme }) => theme.semantic.text.secondary};
`;

export const McapHintAlert = styled(Alert)`
  margin-bottom: ${({ theme }) => theme.spacing[2]}px;
  background: ${({ theme }) => theme.semantic.bg.chrome} !important;
  border-color: ${({ theme }) => theme.semantic.border.accent} !important;

  .ant-alert-message,
  .ant-alert-description,
  .ant-alert-icon {
    color: ${({ theme }) => theme.semantic.text.primary} !important;
  }

  .ant-alert-icon {
    color: ${({ theme }) => theme.semantic.accent.default} !important;
  }
`;
