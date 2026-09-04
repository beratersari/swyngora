import styled from 'styled-components';

export const PageStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[5]}px;
  min-width: 0;
`;

export const Toolbar = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[3]}px;
  align-items: flex-end;
`;

export const Field = styled.label`
  display: flex;
  flex-direction: column;
  gap: 4px;
`;

export const DatetimeInput = styled.input`
  min-width: 210px;
  height: 32px;
  padding: 4px 11px;
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.md}px;
  background: ${({ theme }) => theme.semantic.bg.surface};
  color: ${({ theme }) => theme.semantic.text.primary};
  font: inherit;
`;

export const EmptyHint = styled.div`
  padding: ${({ theme }) => theme.spacing[3]}px ${({ theme }) => theme.spacing[4]}px;
  background: ${({ theme }) => theme.semantic.bg.muted};
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.md}px;
  border-left: 3px solid ${({ theme }) => theme.semantic.accent.default};
`;

export const ErrorHint = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[1]}px;
  padding: ${({ theme }) => theme.spacing[3]}px ${({ theme }) => theme.spacing[4]}px;
  background: ${({ theme }) => theme.semantic.bg.muted};
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.md}px;
  border-left: 3px solid ${({ theme }) => theme.semantic.status.error};
`;

export const SummaryBlock = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;
