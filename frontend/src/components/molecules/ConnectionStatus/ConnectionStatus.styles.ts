import styled, { css, keyframes } from 'styled-components';
import type { ConnectionStatusKind } from './ConnectionStatus.types';

const livePulse = keyframes`
  0% { box-shadow: 0 0 0 0 rgba(0, 255, 129, 0.45); }
  70% { box-shadow: 0 0 0 8px rgba(0, 255, 129, 0); }
  100% { box-shadow: 0 0 0 0 rgba(0, 255, 129, 0); }
`;

export const Pill = styled.div`
  display: inline-flex;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[2]}px;
  padding: 4px 10px;
  border-radius: ${({ theme }) => theme.radii.pill}px;
  border: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  background: ${({ theme }) => theme.semantic.bg.canvas};
  white-space: nowrap;

  ${({ theme }) => theme.media.phone} {
    padding: 4px 6px;
  }
`;

export const Dot = styled.span<{ $status: ConnectionStatusKind }>`
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: ${({ theme, $status }) => {
    if ($status === 'live') return theme.semantic.chart.up;
    if ($status === 'degraded') return theme.semantic.status.warning;
    if ($status === 'offline') return theme.semantic.status.error;
    if ($status === 'paused') return theme.semantic.status.warning;
    return theme.semantic.text.tertiary;
  }};
  box-shadow: ${({ theme, $status }) =>
    $status === 'live' ? `0 0 0 3px ${theme.semantic.accent.soft}` : 'none'};
  ${({ $status }) =>
    $status === 'live'
      ? css`
          animation: ${livePulse} 1.8s ease-out infinite;
        `
      : null};

  @media (prefers-reduced-motion: reduce) {
    animation: none;
  }
`;

export const Label = styled.span`
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.02em;
  color: ${({ theme }) => theme.semantic.text.primary};

  ${({ theme }) => theme.media.phone} {
    display: none;
  }
`;
