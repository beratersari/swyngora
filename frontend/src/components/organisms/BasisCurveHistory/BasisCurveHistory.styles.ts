import styled from 'styled-components';

export const Stack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
  min-width: 0;
`;

export const Chips = styled.div`
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: ${({ theme }) => theme.spacing[2]}px;

  @media (max-width: 720px) {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
`;

export const Chip = styled.div`
  min-width: 0;
  padding: ${({ theme }) => theme.spacing[2]}px ${({ theme }) => theme.spacing[3]}px;
  border-radius: ${({ theme }) => theme.radii.sm}px;
  border: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  background: ${({ theme }) => theme.semantic.bg.muted};
`;

export const Toolbar = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const ChartBox = styled.div`
  width: 100%;
  min-width: 0;
  min-height: 180px;
  height: 200px;
  border-radius: ${({ theme }) => theme.radii.sm}px;
  border: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  background: ${({ theme }) => theme.semantic.bg.page};
  overflow: hidden;
`;

export const Moves = styled.ul`
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 6px;
`;
