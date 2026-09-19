import styled from 'styled-components';

export const Panel = styled.section`
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
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

export const VenueCard = styled.article`
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
  padding: 14px 16px;
  border-radius: 10px;
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  background: ${({ theme }) => theme.semantic.bg.canvas};
`;

export const VenueHead = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
  font-size: 14px;
  font-weight: 700;
`;

export const Chip = styled.span<{ $tone?: 'premium' | 'discount' | 'flat' | 'contango' | 'backwardation' | 'humped' }>`
  display: inline-flex;
  align-items: center;
  padding: 1px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
  color: ${({ theme, $tone }) =>
    $tone === 'premium' || $tone === 'contango'
      ? theme.semantic.status.success
      : $tone === 'discount' || $tone === 'backwardation'
        ? theme.semantic.status.error
        : theme.semantic.text.secondary};
  background: ${({ theme }) => theme.semantic.bg.elevated};
`;

export const CurveBanner = styled.div<{ $tone: 'contango' | 'backwardation' | 'flat' | 'humped' }>`
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 14px;
  border-radius: 10px;
  border: 1px solid
    ${({ theme, $tone }) =>
      $tone === 'contango'
        ? theme.semantic.border.accent
        : $tone === 'backwardation'
          ? theme.semantic.border.danger
          : theme.semantic.border.default};
  background: ${({ theme, $tone }) =>
    $tone === 'contango'
      ? theme.semantic.bg.accentSoft
      : $tone === 'backwardation'
        ? theme.semantic.bg.dangerSoft
        : theme.semantic.bg.canvas};
`;

export const CurveTable = styled.div`
  display: grid;
  grid-template-columns: minmax(88px, 1.2fr) 72px 56px minmax(72px, 1fr) minmax(88px, 1fr);
  gap: 8px 10px;
  align-items: center;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  overflow-x: auto;
`;

export const Head = styled.span`
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: ${({ theme }) => theme.semantic.text.tertiary};
`;

export const Cell = styled.span<{ $hot?: boolean }>`
  font-weight: ${({ $hot }) => ($hot ? 700 : 500)};
  color: ${({ theme, $hot }) => ($hot ? theme.semantic.text.primary : theme.semantic.text.secondary)};
`;
