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

export const Stats = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  font-size: 12px;
  color: ${({ theme }) => theme.semantic.text.secondary};

  strong {
    color: ${({ theme }) => theme.semantic.text.primary};
    font-variant-numeric: tabular-nums;
  }
`;

export const Frame = styled.div`
  position: relative;
  min-width: 0;
  flex: 1 1 auto;
  min-height: 420px;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: 8px;
  overflow: hidden;

  &:fullscreen {
    min-height: 100%;
    height: 100%;
    border: 0;
    border-radius: 0;
  }
`;

export const Plot = styled.svg`
  display: block;
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
`;

export const Tip = styled.div<{ $x: number; $y: number }>`
  position: absolute;
  left: ${({ $x }) => $x}px;
  top: ${({ $y }) => $y}px;
  z-index: 6;
  min-width: 168px;
  padding: 10px 12px;
  border-radius: 8px;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  color: ${({ theme }) => theme.semantic.text.primary};
  border: 1px solid ${({ theme }) => theme.semantic.border.strong};
  font-size: 12px;
  pointer-events: none;
  box-shadow: 0 10px 28px rgba(13, 20, 33, 0.14);
`;
