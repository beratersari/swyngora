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

export const ChipRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const Chip = styled.button<{ $active?: boolean }>`
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

export const ScaleLegend = styled.div`
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
`;

export const ScaleBar = styled.span<{ $tone: 'totals' | 'longs' | 'shorts' }>`
  display: inline-block;
  width: 88px;
  height: 10px;
  border-radius: 999px;
  background: ${({ $tone }) =>
    $tone === 'longs'
      ? 'linear-gradient(90deg, #481616 0%, #ea3943 100%)'
      : $tone === 'shorts'
        ? 'linear-gradient(90deg, #0a382a 0%, #16c784 100%)'
        : 'linear-gradient(90deg, #3e300c 0%, #e88c12 55%, #ec3048 100%)'};
`;

export const MapFrame = styled.div`
  position: relative;
  width: 100%;
  height: 360px;
  min-height: 280px;
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.md}px;
  background: #0b1220;
  overflow: hidden;

  @media (min-width: 1100px) {
    height: 440px;
  }
`;

export const HeatCanvas = styled.canvas`
  display: block;
  width: 100%;
  height: 100%;
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
  max-height: 360px;
  overflow: auto;
  border: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  border-radius: ${({ theme }) => theme.radii.sm}px;
`;

export const SignalTable = styled.table`
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;

  th,
  td {
    padding: 6px 8px;
    text-align: left;
    border-bottom: 1px solid ${({ theme }) => theme.semantic.border.subtle};
    white-space: nowrap;
    vertical-align: top;
  }

  th {
    position: sticky;
    top: 0;
    background: ${({ theme }) => theme.semantic.bg.canvas};
    color: ${({ theme }) => theme.semantic.text.secondary};
    font-weight: 600;
    z-index: 1;
  }

  td[data-gap='true'] {
    color: ${({ theme }) => theme.semantic.text.secondary};
  }
`;

export const HoverCard = styled.div<{ $x: number; $y: number }>`
  position: absolute;
  left: ${({ $x }) => $x}px;
  top: ${({ $y }) => $y}px;
  z-index: 2;
  min-width: 168px;
  padding: 8px 10px;
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.sm}px;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  box-shadow: 0 8px 24px rgb(13 20 33 / 16%);
  pointer-events: none;
`;
