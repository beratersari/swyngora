import styled from 'styled-components';

export const ChartContainer = styled.div<{ $height: number }>`
  width: 100%;
  height: ${({ $height }) => $height}px;
  border-radius: ${({ theme }) => theme.radii.md}px;
  overflow: hidden;
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  background: ${({ theme }) => theme.semantic.bg.canvas};
`;
