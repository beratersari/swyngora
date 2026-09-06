import styled from 'styled-components';

export const Stack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
  min-width: 0;
`;

export const Card = styled.article<{ $notable?: boolean }>`
  min-width: 0;
  padding: ${({ theme }) => theme.spacing[2]}px ${({ theme }) => theme.spacing[3]}px;
  border-radius: ${({ theme }) => theme.radii.sm}px;
  border: 1px solid
    ${({ theme, $notable }) =>
      $notable ? theme.semantic.status.warning : theme.semantic.border.subtle};
  background: ${({ theme, $notable }) =>
    $notable ? theme.semantic.bg.accentSoft : theme.semantic.bg.muted};
`;

export const Topic = styled.span`
  display: inline-block;
  margin-right: ${({ theme }) => theme.spacing[2]}px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
`;
