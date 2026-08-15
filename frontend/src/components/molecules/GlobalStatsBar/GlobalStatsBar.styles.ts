import styled from 'styled-components';

export const Bar = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 20px;
  max-width: 1280px;
  margin: 0 auto;
  padding: 10px 24px;
  font-size: 12px;
  color: ${({ theme }) => theme.semantic.text.secondary};
`;

export const Stat = styled.span`
  display: inline-flex;
  align-items: baseline;
  gap: 6px;
  white-space: nowrap;
`;

export const StatLabel = styled.span`
  color: ${({ theme }) => theme.semantic.text.tertiary};
  font-weight: 500;
`;

export const StatValue = styled.span<{ $tone?: 'up' | 'down' | 'accent' }>`
  font-weight: 700;
  color: ${({ theme, $tone }) => {
    if ($tone === 'up') return theme.semantic.chart.up;
    if ($tone === 'down') return theme.semantic.chart.down;
    if ($tone === 'accent') return theme.semantic.text.primary;
    return theme.semantic.text.primary;
  }};
`;
