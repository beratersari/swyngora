import styled from 'styled-components';

export const Panel = styled.section`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
  padding: ${({ theme }) => theme.spacing[3]}px;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  border: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  border-radius: ${({ theme }) => theme.radii.md}px;
  min-width: 0;
`;

export const TitleRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const WindowRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const WindowChip = styled.button<{ $active?: boolean }>`
  margin: 0;
  padding: 4px 10px;
  border: 1px solid
    ${({ theme, $active }) =>
      $active ? theme.semantic.accent.default : theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.sm}px;
  background: ${({ theme, $active }) =>
    $active ? theme.semantic.bg.accentSoft : theme.semantic.bg.muted};
  color: ${({ theme }) => theme.semantic.text.primary};
  font-family: ${({ theme }) => theme.fontFamilies.sans};
  font-size: 12px;
  cursor: pointer;

  &:hover {
    border-color: ${({ theme }) => theme.semantic.accent.default};
  }
`;

export const LegendRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[3]}px;
  align-items: center;
`;

export const ScaleLegend = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
`;

export const ScaleBar = styled.span`
  display: inline-block;
  width: 88px;
  height: 10px;
  border-radius: 999px;
  background: linear-gradient(
    90deg,
    ${({ theme }) => theme.semantic.chart.up} 0%,
    ${({ theme }) => theme.semantic.bg.page} 50%,
    ${({ theme }) => theme.semantic.chart.down} 100%
  );
`;

export const MapFrame = styled.div`
  position: relative;
  width: 100%;
  height: 320px;
  min-height: 260px;
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.md}px;
  background: ${({ theme }) => theme.semantic.bg.page};
  overflow: hidden;

  @media (min-width: 1100px) {
    height: 400px;
  }
`;

export const HeatCanvas = styled.canvas`
  display: block;
  width: 100%;
  height: 100%;
`;

export const HoverCard = styled.div<{ $x: number; $y: number }>`
  position: absolute;
  left: ${({ $x }) => $x}px;
  top: ${({ $y }) => $y}px;
  z-index: 2;
  min-width: 160px;
  padding: 8px 10px;
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.sm}px;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  box-shadow: 0 8px 24px rgb(13 20 33 / 12%);
  pointer-events: none;
`;
