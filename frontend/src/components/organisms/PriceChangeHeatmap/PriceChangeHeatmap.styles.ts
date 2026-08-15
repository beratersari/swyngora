import styled from 'styled-components';
import { withAlpha } from '@/styles/tokens';

export const Shell = styled.div`
  display: grid;
  grid-template-columns: minmax(0, 1fr) 280px;
  grid-template-rows: minmax(0, 1fr);
  min-width: 0;
  min-height: 0;
  flex: 1 1 auto;
  width: 100%;
  height: 100%;
  background: ${({ theme }) => theme.semantic.bg.canvas};

  ${({ theme }) => theme.media.tablet} {
    grid-template-columns: minmax(0, 1fr);
    grid-template-rows: minmax(240px, 1fr) auto;
  }
`;

export const Board = styled.div`
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr);
  min-width: 0;
  min-height: 0;
  height: 100%;
`;

export const Scale = styled.div`
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: stretch;
  gap: 6px;
  padding: 10px 0;
  border-right: 1px solid ${({ theme }) => theme.semantic.border.subtle};
`;

export const ScaleBar = styled.div`
  flex: 1;
  width: 8px;
  min-height: 80px;
`;

export const ScaleTick = styled.span`
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-size: 9px;
  letter-spacing: 0.04em;
  color: ${({ theme }) => theme.semantic.text.tertiary};
  writing-mode: horizontal-tb;
`;

export const MapFrame = styled.div`
  position: relative;
  width: 100%;
  min-width: 0;
  min-height: 320px;
  height: 100%;
  background: ${({ theme }) => theme.semantic.chart.mapBed};
  overflow: hidden;
`;

export const TileHost = styled.div<{ $x: number; $y: number; $w: number; $h: number }>`
  position: absolute;
  left: ${({ $x }) => $x}%;
  top: ${({ $y }) => $y}%;
  width: ${({ $w }) => $w}%;
  height: ${({ $h }) => $h}%;
  padding: 1px;
  box-sizing: border-box;
`;

export const TileButton = styled.button<{ $fill: string; $on?: boolean }>`
  width: 100%;
  height: 100%;
  margin: 0;
  padding: 0 4px;
  border: 0;
  border-radius: 0;
  background: ${({ $fill }) => $fill};
  color: ${({ theme }) => theme.semantic.text.primary};
  text-shadow: none;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  justify-content: flex-end;
  gap: 1px;
  overflow: hidden;
  text-align: left;
  box-sizing: border-box;
  border-radius: 6px;
  box-shadow: ${({ theme, $on }) =>
    $on ? `inset 0 0 0 2px ${theme.semantic.accent.default}` : 'none'};

  &:hover {
    z-index: 2;
    filter: brightness(1.08);
  }

  &:focus-visible {
    z-index: 3;
    outline: 2px solid ${({ theme }) => theme.semantic.border.focus};
    outline-offset: -2px;
  }
`;

export const TileSymbol = styled.span<{ $size: 'full' | 'compact' | 'ticker' | 'micro' }>`
  font-family: ${({ theme }) => theme.fontFamilies.sans};
  font-weight: ${({ theme }) => theme.fontWeights.bold};
  font-size: ${({ $size }) => ($size === 'full' ? 13 : $size === 'compact' ? 11 : 10)}px;
  letter-spacing: 0.04em;
  line-height: 1.1;
  text-transform: uppercase;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
`;

export const TileChange = styled.span`
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-variant-numeric: tabular-nums;
  font-weight: ${({ theme }) => theme.fontWeights.semibold};
  font-size: 11px;
  line-height: 1.15;
`;

export const TilePrice = styled.span`
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-variant-numeric: tabular-nums;
  font-size: 10px;
  line-height: 1.15;
  opacity: 0.72;
`;

export const Inspector = styled.aside`
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
  padding: 16px;
  border-left: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  background: ${({ theme }) => theme.semantic.bg.page};

  ${({ theme }) => theme.media.tablet} {
    border-left: 0;
    border-top: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  }
`;

export const InspectorKicker = styled.p`
  margin: 0;
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: ${({ theme }) => theme.semantic.text.tertiary};
`;

export const InspectorPair = styled.h2`
  margin: 0;
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.02em;
  color: ${({ theme }) => theme.semantic.text.primary};
`;

export const InspectorChange = styled.p<{ $up?: boolean; $flat?: boolean }>`
  margin: 0;
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-variant-numeric: tabular-nums;
  font-size: 28px;
  font-weight: 600;
  color: ${({ theme, $up, $flat }) => {
    if ($flat) return theme.semantic.text.secondary;
    return $up ? theme.semantic.chart.up : theme.semantic.chart.down;
  }};
`;

export const InspectorMeta = styled.dl`
  margin: 0;
  display: grid;
  gap: 8px;
`;

export const InspectorRow = styled.div`
  display: flex;
  justify-content: space-between;
  gap: 12px;
  font-size: 12px;
  border-bottom: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  padding-bottom: 6px;
`;

export const InspectorDt = styled.dt`
  color: ${({ theme }) => theme.semantic.text.tertiary};
  text-transform: uppercase;
  letter-spacing: 0.06em;
  font-size: 10px;
`;

export const InspectorDd = styled.dd`
  margin: 0;
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-variant-numeric: tabular-nums;
  color: ${({ theme }) => theme.semantic.text.primary};
`;

export const InspectorHint = styled.p`
  margin: auto 0 0;
  font-size: 12px;
  color: ${({ theme }) => theme.semantic.text.tertiary};
`;

/** Unused HUD aliases — kept if any test imported them. */
export const Hud = styled.div`
  display: none;
`;
export const HudPair = styled.div``;
export const HudChange = styled.div``;
export const HudMeta = styled.div``;
