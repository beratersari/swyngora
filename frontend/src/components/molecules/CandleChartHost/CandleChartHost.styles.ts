import styled from 'styled-components';

export const ChartShell = styled.div<{ $height: number }>`
  position: relative;
  width: 100%;
  height: ${({ $height }) => $height}px;
  /* Contain layout so sibling reflows don't thrash the canvas size. */
  contain: layout style;
`;

export const ChartContainer = styled.div`
  width: 100%;
  height: 100%;
  /* Chart library attaches pointer handlers here; keep it interactive. */
  touch-action: none;
`;

export const ChartSkeletonLayer = styled.div`
  position: absolute;
  inset: 0;
  z-index: 1;
  pointer-events: none;
`;
