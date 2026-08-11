import { Link } from 'react-router-dom';
import styled, { keyframes } from 'styled-components';

const scroll = keyframes`
  from { transform: translate3d(0, 0, 0); }
  to { transform: translate3d(-50%, 0, 0); }
`;

export const Track = styled.div`
  display: flex;
  overflow: hidden;
  contain: content;
  border-bottom: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  background: ${({ theme }) => theme.semantic.bg.canvas};
`;

export const Strip = styled.div<{ $paused?: boolean }>`
  display: flex;
  width: max-content;
  will-change: transform;
  animation: ${scroll} 48s linear infinite;
  animation-play-state: ${({ $paused }) => ($paused ? 'paused' : 'running')};

  @media (prefers-reduced-motion: reduce) {
    animation: none;
    will-change: auto;
  }

  &:hover {
    animation-play-state: paused;
  }
`;

export const Cell = styled(Link)`
  display: inline-flex;
  align-items: baseline;
  gap: 8px;
  padding: 6px 16px;
  text-decoration: none;
  color: ${({ theme }) => theme.semantic.text.primary};
  border-right: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  white-space: nowrap;
  transition: background ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard};

  &:hover {
    background: ${({ theme }) => theme.semantic.bg.hover};
    color: ${({ theme }) => theme.semantic.text.primary};
  }
`;

export const Sym = styled.span`
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-size: 12px;
  font-weight: 600;
`;

export const Px = styled.span`
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-size: 12px;
  color: ${({ theme }) => theme.semantic.text.primary};
`;

export const Chg = styled.span<{ $up: boolean; $flat: boolean }>`
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-size: 12px;
  font-weight: 600;
  color: ${({ theme, $up, $flat }) => {
    if ($flat) return theme.semantic.text.secondary;
    return $up ? theme.semantic.chart.up : theme.semantic.chart.down;
  }};
`;
