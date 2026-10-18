import styled from 'styled-components';

export const Panel = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
  padding: ${({ theme }) => theme.spacing[3]}px;
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.md}px;
  background: ${({ theme }) => theme.semantic.bg.chrome};
  min-width: 0;
`;

export const Status = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[2]}px;
  min-width: 0;
`;

export const Columns = styled.div<{ $busy?: boolean }>`
  opacity: ${({ $busy }) => ($busy ? 0.55 : 1)};
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: ${({ theme }) => theme.spacing[2]}px;
  min-width: 0;
`;

export const Card = styled.div<{ $selected: boolean }>`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
  min-width: 0;
  padding: ${({ theme }) => theme.spacing[2]}px;
  border: 1px solid
    ${({ theme, $selected }) =>
      $selected ? theme.semantic.border.focus : theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.md}px;
  background: ${({ theme }) => theme.semantic.bg.canvas};
`;

export const Pick = styled.button`
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  width: 100%;
  margin: 0;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  text-align: left;
  cursor: pointer;
`;

export const Line = styled.div`
  display: flex;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[2]}px;
  width: 100%;
  min-width: 0;
`;

export const Schedule = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;
