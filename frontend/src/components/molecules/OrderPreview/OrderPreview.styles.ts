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

export const Grid = styled.dl`
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 4px ${({ theme }) => theme.spacing[3]}px;
  margin: 0;
`;

export const Label = styled.dt`
  margin: 0;
  min-width: 0;
`;

export const Value = styled.dd`
  margin: 0;
  text-align: right;
`;
