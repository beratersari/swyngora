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

export const Swatch = styled.span<{ $tone?: 'long' | 'short' | 'last' | 'line'; $color?: string }>`
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: ${({ theme }) => theme.semantic.text.secondary};

  &::before {
    content: '';
    width: ${({ $tone }) => ($tone === 'line' ? '14px' : '10px')};
    height: ${({ $tone }) => ($tone === 'line' ? '2px' : '10px')};
    border-radius: ${({ $tone }) => ($tone === 'line' ? '1px' : '2px')};
    background: ${({ $tone, $color, theme }) =>
      $color
        ? $color
        : $tone === 'long'
          ? theme.semantic.chart.down
          : $tone === 'short'
            ? theme.semantic.chart.up
            : $tone === 'line'
              ? theme.semantic.text.primary
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

export const ReviewTable = styled.table`
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;

  th,
  td {
    padding: 5px 8px;
    text-align: right;
    border-bottom: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  }

  th:first-child,
  td:first-child {
    text-align: left;
  }

  th {
    color: ${({ theme }) => theme.semantic.text.secondary};
    font-weight: 600;
  }
`;

export const SignalScroll = styled.div`
  max-height: 280px;
  overflow: auto;
  border: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  border-radius: 8px;
`;

export const Reliability = styled.span<{ $tone: string }>`
  display: inline-flex;
  align-items: center;
  padding: 1px 7px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
  text-transform: lowercase;
  color: ${({ $tone, theme }) =>
    $tone === 'high'
      ? theme.semantic.chart.up
      : $tone === 'low'
        ? theme.semantic.chart.down
        : theme.semantic.text.secondary};
  background: ${({ $tone, theme }) =>
    $tone === 'high'
      ? 'rgba(22, 199, 132, 0.12)'
      : $tone === 'low'
        ? 'rgba(234, 57, 67, 0.12)'
        : theme.semantic.bg.muted};
`;
