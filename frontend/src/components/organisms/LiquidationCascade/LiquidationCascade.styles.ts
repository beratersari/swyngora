import styled from 'styled-components';

export const Panel = styled.section`
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
`;

export const Banner = styled.div<{ $tone: 'quiet' | 'elevated' | 'cascade' | 'extreme' }>`
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 14px 16px;
  border-radius: 10px;
  border: 1px solid
    ${({ theme, $tone }) =>
      $tone === 'extreme' || $tone === 'cascade'
        ? theme.semantic.border.danger
        : $tone === 'elevated'
          ? theme.semantic.border.accent
          : theme.semantic.border.default};
  background: ${({ theme, $tone }) =>
    $tone === 'extreme' || $tone === 'cascade'
      ? theme.semantic.bg.dangerSoft
      : $tone === 'elevated'
        ? theme.semantic.bg.accentSoft
        : theme.semantic.bg.canvas};
`;

export const BannerTitle = styled.div`
  font-size: 16px;
  font-weight: 700;
  color: ${({ theme }) => theme.semantic.text.primary};
`;

export const BothNote = styled.div`
  font-size: 13px;
  font-weight: 600;
  color: ${({ theme }) => theme.semantic.status.error};
`;

export const VenueGrid = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;

  @media (max-width: 860px) {
    grid-template-columns: 1fr;
  }
`;

export const VenueCard = styled.article<{ $tone: 'quiet' | 'elevated' | 'cascade' | 'extreme' }>`
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
  padding: 14px;
  border-radius: 10px;
  border: 1px solid
    ${({ theme, $tone }) =>
      $tone === 'quiet' ? theme.semantic.border.default : theme.semantic.border.strong};
  background: ${({ theme }) => theme.semantic.bg.canvas};
`;

export const VenueHead = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
`;

export const GradeChip = styled.span<{ $tone: 'quiet' | 'elevated' | 'cascade' | 'extreme' }>`
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: ${({ theme, $tone }) =>
    $tone === 'extreme' || $tone === 'cascade'
      ? theme.semantic.status.error
      : $tone === 'elevated'
        ? theme.semantic.status.warning
        : theme.semantic.text.secondary};
  background: ${({ theme, $tone }) =>
    $tone === 'extreme' || $tone === 'cascade'
      ? theme.semantic.bg.dangerSoft
      : $tone === 'elevated'
        ? theme.semantic.bg.accentSoft
        : theme.semantic.bg.muted};
`;

export const SideChip = styled.span<{ $tone: 'long' | 'short' | 'both' | 'none' }>`
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  color: ${({ theme, $tone }) =>
    $tone === 'long'
      ? theme.semantic.chart.down
      : $tone === 'short'
        ? theme.semantic.chart.up
        : theme.semantic.text.secondary};
`;

export const WindowTable = styled.div`
  display: grid;
  grid-template-columns: 48px 1fr 1fr 56px 64px;
  gap: 6px 8px;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
`;

export const WindowHead = styled.span`
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: ${({ theme }) => theme.semantic.text.tertiary};
`;

export const WindowCell = styled.span<{ $hot?: boolean }>`
  font-weight: ${({ $hot }) => ($hot ? 700 : 500)};
  color: ${({ theme, $hot }) =>
    $hot ? theme.semantic.text.primary : theme.semantic.text.secondary};
`;

export const HitsWrap = styled.div`
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
`;

export const HitsTable = styled.div`
  display: grid;
  grid-template-columns: minmax(88px, 1fr) 72px 88px 64px 72px 56px;
  gap: 8px 10px;
  align-items: center;
  font-size: 13px;
  font-variant-numeric: tabular-nums;
`;

export const HitButton = styled.button`
  margin: 0;
  padding: 0;
  border: 0;
  background: none;
  text-align: left;
  font: inherit;
  font-weight: 700;
  color: ${({ theme }) => theme.semantic.text.link};
  cursor: pointer;

  &:hover {
    color: ${({ theme }) => theme.semantic.text.linkHover};
  }
`;
