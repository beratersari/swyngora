import styled from 'styled-components';

export const Panel = styled.section`
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
`;

export const Banner = styled.div`
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 14px 16px;
  border-radius: 10px;
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  background: ${({ theme }) => theme.semantic.bg.canvas};
`;

export const BannerTitle = styled.div`
  font-size: 16px;
  font-weight: 700;
  color: ${({ theme }) => theme.semantic.text.primary};
`;

export const VenueStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
`;

export const VenueBlock = styled.article`
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
`;

export const VenueHead = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px 16px;
`;

export const VenueMeta = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
`;

export const CompareGrid = styled.div`
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;

  @media (max-width: 860px) {
    grid-template-columns: 1fr;
  }
`;

export const SideCard = styled.div<{ $side: 'up' | 'down' }>`
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
  padding: 14px;
  border-radius: 10px;
  border: 1px solid
    ${({ theme, $side }) =>
      $side === 'up' ? theme.semantic.status.success : theme.semantic.border.danger};
  background: ${({ theme }) => theme.semantic.bg.canvas};
`;

export const SideHead = styled.div`
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
`;

export const PriceNum = styled.span`
  font-size: 26px;
  font-weight: 800;
  line-height: 1;
  letter-spacing: -0.03em;
  font-variant-numeric: tabular-nums;
  color: ${({ theme }) => theme.semantic.text.primary};
`;

export const SideChip = styled.span<{ $side: 'up' | 'down' }>`
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.02em;
  text-transform: uppercase;
  color: ${({ theme }) => theme.semantic.text.primary};
  background: ${({ theme, $side }) =>
    $side === 'up' ? theme.semantic.bg.successSoft : theme.semantic.bg.dangerSoft};
`;

export const MetricTable = styled.dl`
  display: grid;
  grid-template-columns: minmax(92px, 36%) 1fr;
  gap: 6px 10px;
  margin: 0;
`;

export const MetricLabel = styled.dt`
  margin: 0;
  font-size: 12px;
  color: ${({ theme }) => theme.semantic.text.secondary};
`;

export const MetricValue = styled.dd`
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: ${({ theme }) => theme.semantic.text.primary};
`;

export const ExtraList = styled.ul`
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
`;

export const ExtraRow = styled.li`
  display: flex;
  justify-content: space-between;
  gap: 10px;
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  color: ${({ theme }) => theme.semantic.text.secondary};
`;

export const Hint = styled.p`
  margin: 0;
  font-size: 12px;
  line-height: 1.45;
  color: ${({ theme }) => theme.semantic.text.secondary};
`;
