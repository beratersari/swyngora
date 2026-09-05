import styled from 'styled-components';

export const Stack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;
`;

export const Cards = styled.div`
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: ${({ theme }) => theme.spacing[2]}px;

  @media (max-width: 720px) {
    grid-template-columns: 1fr;
  }
`;

export const Card = styled.div`
  min-width: 0;
  padding: ${({ theme }) => theme.spacing[2]}px ${({ theme }) => theme.spacing[3]}px;
  border-radius: ${({ theme }) => theme.radii.sm}px;
  border: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  background: ${({ theme }) => theme.semantic.bg.muted};
`;

export const StrikeRow = styled.div<{ $maxPain?: boolean }>`
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr) 56px;
  gap: ${({ theme }) => theme.spacing[2]}px;
  align-items: center;
  padding: 2px 0;
  background: ${({ theme, $maxPain }) =>
    $maxPain ? theme.semantic.bg.accentSoft : 'transparent'};
`;

export const Track = styled.div`
  height: 8px;
  border-radius: 4px;
  background: ${({ theme }) => theme.semantic.bg.page};
  overflow: hidden;
`;

export const Fill = styled.div<{ $width: number }>`
  height: 100%;
  width: ${({ $width }) => `${$width}%`};
  background: ${({ theme }) => theme.semantic.accent.default};
`;
