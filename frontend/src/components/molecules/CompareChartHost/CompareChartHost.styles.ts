import styled from 'styled-components';

export const ChartShell = styled.div<{ $height: number }>`
  position: relative;
  width: 100%;
  height: ${({ $height }) => $height}px;
`;

export const ChartContainer = styled.div<{ $height: number; $hidden?: boolean }>`
  width: 100%;
  height: ${({ $height }) => $height}px;
  border-radius: ${({ theme }) => theme.radii.md}px;
  overflow: hidden;
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  background: ${({ theme }) => theme.semantic.bg.canvas};
  visibility: ${({ $hidden }) => ($hidden ? 'hidden' : 'visible')};
`;

export const ChartSkeletonLayer = styled.div`
  position: absolute;
  inset: 0;
  z-index: 1;
`;
