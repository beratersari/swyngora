import styled from 'styled-components';

export const PageStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[4]}px;
  min-height: min(70vh, 720px);
`;

export const PageIntro = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[1]}px;
`;

export const Thread = styled.div`
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-height: 280px;
  max-height: min(55vh, 520px);
  overflow-y: auto;
  padding: ${({ theme }) => theme.spacing[3]}px;
  border-radius: 12px;
  border: 1px solid ${({ theme }) => theme.colors.bangladeshGreen}55;
  background: ${({ theme }) => theme.colors.pine}88;
`;

export const BubbleRow = styled.div<{ $role: 'user' | 'assistant' | 'system' }>`
  display: flex;
  justify-content: ${({ $role }) => ($role === 'user' ? 'flex-end' : 'flex-start')};
`;

export const Bubble = styled.div<{ $role: 'user' | 'assistant' | 'system'; $error?: boolean }>`
  max-width: min(720px, 92%);
  padding: ${({ theme }) => `${theme.spacing[2]}px ${theme.spacing[3]}px`};
  border-radius: 12px;
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.45;
  border: 1px solid
    ${({ theme, $error, $role }) =>
      $error
        ? '#c45c5c'
        : $role === 'user'
          ? theme.colors.mountainMeadow + '66'
          : theme.colors.basil};
  background: ${({ theme, $role, $error }) =>
    $error
      ? '#3a1a1a'
      : $role === 'user'
        ? theme.colors.forest
        : theme.colors.darkGreen};
  color: ${({ theme }) => theme.colors.antiFlashWhite};
`;

export const MetaRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[1]}px;
  margin-top: ${({ theme }) => theme.spacing[2]}px;
`;

export const Composer = styled.form`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const ComposerRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[2]}px;
  align-items: flex-end;
`;

export const Suggestions = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const EmptyState = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  align-items: flex-start;
  padding: ${({ theme }) => theme.spacing[2]}px 0;
`;

export const ToolbarRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[2]}px;
  align-items: center;
`;
