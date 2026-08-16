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

export const MetricRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const MetricChip = styled.button<{ $active?: boolean }>`
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

export const ChartFrame = styled.div`
  position: relative;
  width: 100%;
  height: 280px;
  min-height: 220px;
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.md}px;
  background: ${({ theme }) => theme.semantic.bg.page};
  overflow: hidden;
`;

export const DepthCanvas = styled.canvas`
  display: block;
  width: 100%;
  height: 100%;
`;

export const HoverCard = styled.div<{ $x: number; $y: number }>`
  position: absolute;
  left: ${({ $x }) => $x}px;
  top: ${({ $y }) => $y}px;
  z-index: 2;
  min-width: 148px;
  padding: 8px 10px;
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.sm}px;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  box-shadow: 0 8px 24px rgb(13 20 33 / 12%);
  pointer-events: none;
`;
