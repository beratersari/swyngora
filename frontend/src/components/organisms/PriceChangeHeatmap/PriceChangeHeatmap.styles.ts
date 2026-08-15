import styled from 'styled-components';

export const Shell = styled.div`
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
  min-height: 0;
  flex: 1 1 auto;
  width: 100%;
  height: 100%;
`;

export const MapFrame = styled.div`
  position: relative;
  width: 100%;
  min-width: 0;
  min-height: 420px;
  height: 100%;
  background: #ffffff;
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: 8px;
  overflow: hidden;

  &:fullscreen {
    min-height: 100%;
    height: 100%;
    border: 0;
    border-radius: 0;
    background: ${({ theme }) => theme.semantic.bg.canvas};
    padding: 12px;
  }
`;

export const TileHost = styled.div<{ $x: number; $y: number; $w: number; $h: number }>`
  position: absolute;
  left: ${({ $x }) => $x}%;
  top: ${({ $y }) => $y}%;
  width: ${({ $w }) => $w}%;
  height: ${({ $h }) => $h}%;
  padding: 0;
  box-sizing: border-box;

  &:hover,
  &:focus-within {
    z-index: 2;
  }
`;

export const TileButton = styled.button<{ $fill: string; $ink: string }>`
  width: 100%;
  height: 100%;
  margin: 0;
  padding: 2px 4px;
  border: 0;
  border-radius: 2px;
  background: ${({ $fill }) => $fill};
  color: ${({ $ink }) => $ink};
  text-shadow: 0 1px 1px rgba(13, 20, 33, 0.18);
  cursor: pointer;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 1px;
  overflow: hidden;
  text-align: center;
  box-sizing: border-box;
  user-select: none;

  &:hover {
    filter: brightness(1.06);
  }

  &:focus-visible {
    outline: 2px solid ${({ theme }) => theme.semantic.border.focus};
    outline-offset: -2px;
  }
`;

export const TileSymbol = styled.span<{ $size: 'full' | 'compact' | 'ticker' | 'micro' }>`
  font-family: ${({ theme }) => theme.fontFamilies.sans};
  font-weight: 700;
  font-size: ${({ $size }) =>
    $size === 'full' ? 16 : $size === 'compact' ? 13 : $size === 'ticker' ? 11 : 9}px;
  letter-spacing: 0.01em;
  line-height: 1.15;
  text-transform: uppercase;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
`;

export const TileChange = styled.span<{ $size: 'full' | 'compact' }>`
  font-family: ${({ theme }) => theme.fontFamilies.sans};
  font-variant-numeric: tabular-nums;
  font-weight: 600;
  font-size: ${({ $size }) => ($size === 'full' ? 13 : 11)}px;
  line-height: 1.15;
  opacity: 0.96;
`;

export const Legend = styled.div`
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  padding: 0 2px;
`;

export const LegendBar = styled.div`
  width: 160px;
  height: 8px;
  border-radius: 99px;
`;

export const LegendTick = styled.span`
  font-size: 11px;
  font-weight: 600;
  color: ${({ theme }) => theme.semantic.text.tertiary};
  font-variant-numeric: tabular-nums;
`;

export const HoverCard = styled.div<{ $x: number; $y: number }>`
  position: absolute;
  left: ${({ $x }) => $x}px;
  top: ${({ $y }) => $y}px;
  z-index: 6;
  pointer-events: none;
  min-width: 196px;
  padding: 10px 12px;
  border-radius: 8px;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  border: 1px solid ${({ theme }) => theme.semantic.border.strong};
  box-shadow: 0 10px 28px rgba(13, 20, 33, 0.14);
`;

export const TipHead = styled.div`
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
`;

export const TipSym = styled.span`
  font-weight: 700;
  font-size: 14px;
  letter-spacing: 0.02em;
  color: ${({ theme }) => theme.semantic.text.primary};
`;

export const TipChg = styled.span<{ $up?: boolean; $flat?: boolean }>`
  font-variant-numeric: tabular-nums;
  font-weight: 700;
  font-size: 13px;
  color: ${({ theme, $up, $flat }) => {
    if ($flat) return theme.semantic.text.secondary;
    return $up ? theme.semantic.chart.up : theme.semantic.chart.down;
  }};
`;

export const TipRow = styled.div`
  display: flex;
  justify-content: space-between;
  gap: 16px;
  font-size: 12px;
  padding: 2px 0;

  > span:first-child {
    color: ${({ theme }) => theme.semantic.text.tertiary};
  }

  > span:last-child {
    font-variant-numeric: tabular-nums;
    font-weight: 600;
    color: ${({ theme }) => theme.semantic.text.primary};
  }
`;
