import styled from 'styled-components';

export const Panel = styled.section`
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
  min-height: 420px;
`;

export const TitleRow = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
`;

export const LegendRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 16px;
`;

export const Swatch = styled.span<{ $tone?: 'long' | 'short' | 'last'; $color?: string }>`
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: ${({ theme }) => theme.semantic.text.secondary};

  &::before {
    content: '';
    width: 10px;
    height: 10px;
    border-radius: 2px;
    background: ${({ $tone, $color, theme }) =>
      $color
        ? $color
        : $tone === 'long'
          ? theme.semantic.chart.down
          : $tone === 'short'
            ? theme.semantic.chart.up
            : theme.semantic.action.primary};
  }
`;

export const MapFrame = styled.div`
  position: relative;
  width: 100%;
  min-height: 420px;
  height: min(560px, calc(100dvh - 320px));
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: 8px;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  overflow: hidden;
`;

export const ChartCanvas = styled.canvas`
  display: block;
  width: 100%;
  height: 100%;
`;

export const HoverCard = styled.div<{ $x: number; $y: number }>`
  position: absolute;
  left: ${({ $x }) => $x}px;
  top: ${({ $y }) => $y}px;
  z-index: 4;
  pointer-events: none;
  min-width: 168px;
  padding: 10px 12px;
  border-radius: 8px;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  border: 1px solid ${({ theme }) => theme.semantic.border.strong};
  box-shadow: 0 10px 28px rgba(13, 20, 33, 0.14);
`;

export const TipTitle = styled.div`
  font-weight: 700;
  font-size: 13px;
  margin-bottom: 6px;
  color: ${({ theme }) => theme.semantic.text.primary};
`;

export const TipRow = styled.div`
  display: flex;
  justify-content: space-between;
  gap: 16px;
  font-size: 12px;
  padding: 2px 0;

  > span:first-child {
    color: ${({ theme }) => theme.semantic.text.secondary};
  }

  > span:last-child {
    font-variant-numeric: tabular-nums;
    font-weight: 600;
    color: ${({ theme }) => theme.semantic.text.primary};
  }
`;
