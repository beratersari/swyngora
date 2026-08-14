import styled from 'styled-components';
import { withAlpha } from '@/styles/tokens';

export const Shell = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
  min-width: 0;
  flex: 1;
`;

export const MapFrame = styled.div`
  position: relative;
  width: 100%;
  flex: 1;
  min-height: 520px;
  height: calc(100dvh - 168px);
  background: ${({ theme }) => theme.semantic.chart.mapBed};
  overflow: hidden;
`;

export const TileHost = styled.div<{ $x: number; $y: number; $w: number; $h: number }>`
  position: absolute;
  left: ${({ $x }) => $x}%;
  top: ${({ $y }) => $y}%;
  width: ${({ $w }) => $w}%;
  height: ${({ $h }) => $h}%;
`;

export const TileButton = styled.button<{ $fill: string }>`
  width: 100%;
  height: 100%;
  margin: 0;
  padding: 0 3px;
  border: 0;
  border-radius: 0;
  background: ${({ $fill }) => $fill};
  color: ${({ theme }) => theme.semantic.text.primary};
  cursor: pointer;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0;
  overflow: hidden;
  text-align: center;
  box-sizing: border-box;
  box-shadow: none;
  transition: box-shadow ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard};

  &:hover {
    z-index: 2;
    box-shadow: inset 0 0 0 1px ${({ theme }) => withAlpha(theme.palette.antiFlashWhite, 0.22)};
  }

  &:focus-visible {
    z-index: 3;
    outline: 2px solid ${({ theme }) => theme.semantic.border.focus};
    outline-offset: -2px;
  }

  @media (prefers-reduced-motion: reduce) {
    transition: none;
  }
`;

export const TileSymbol = styled.span<{ $size: 'full' | 'compact' | 'ticker' | 'micro' }>`
  font-family: ${({ theme }) => theme.fontFamilies.sans};
  font-weight: ${({ theme }) => theme.fontWeights.bold};
  font-size: ${({ $size, theme }) =>
    $size === 'full' ? theme.typeScale.bodySm.fontSize : $size === 'compact' ? 11 : 11}px;
  letter-spacing: ${({ $size }) => ($size === 'micro' ? '0.02em' : '0.03em')};
  line-height: 1.1;
  text-transform: uppercase;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  opacity: 0.94;
`;

export const TileChange = styled.span`
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-variant-numeric: tabular-nums;
  font-weight: ${({ theme }) => theme.fontWeights.medium};
  font-size: 11px;
  line-height: 1.15;
  opacity: 0.9;
`;

export const TilePrice = styled.span`
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-variant-numeric: tabular-nums;
  font-weight: ${({ theme }) => theme.fontWeights.regular};
  font-size: ${({ theme }) => theme.typeScale.overline.fontSize}px;
  line-height: 1.15;
  opacity: 0.62;
`;

export const Hud = styled.div<{ $x: number; $y: number }>`
  position: absolute;
  left: ${({ $x }) => $x}px;
  top: ${({ $y }) => $y}px;
  z-index: 5;
  min-width: 148px;
  padding: 8px 10px;
  pointer-events: none;
  background: ${({ theme }) => withAlpha(theme.palette.richBlack, 0.92)};
  color: ${({ theme }) => theme.semantic.text.primary};
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
`;

export const HudPair = styled.div`
  font-size: ${({ theme }) => theme.typeScale.overline.fontSize}px;
  font-weight: ${({ theme }) => theme.fontWeights.semibold};
  letter-spacing: 0.04em;
  text-transform: uppercase;
  margin-bottom: 4px;
`;

export const HudChange = styled.div`
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-variant-numeric: tabular-nums;
  font-size: ${({ theme }) => theme.typeScale.numeric.fontSize}px;
  font-weight: ${({ theme }) => theme.fontWeights.semibold};
`;

export const HudMeta = styled.div`
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-variant-numeric: tabular-nums;
  font-size: ${({ theme }) => theme.typeScale.overline.fontSize}px;
  color: ${({ theme }) => theme.semantic.text.secondary};
  margin-top: 2px;
`;
