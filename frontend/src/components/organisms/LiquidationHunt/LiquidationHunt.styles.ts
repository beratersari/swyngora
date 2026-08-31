import styled from 'styled-components';

export const Panel = styled.section`
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
`;

export const Banner = styled.div<{ $tone: 'up' | 'down' | 'even' }>`
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 14px 16px;
  border-radius: 10px;
  border: 1px solid
    ${({ theme, $tone }) =>
      $tone === 'up'
        ? theme.semantic.status.success
        : $tone === 'down'
          ? theme.semantic.border.danger
          : theme.semantic.border.default};
  background: ${({ theme, $tone }) =>
    $tone === 'up'
      ? theme.semantic.bg.successSoft
      : $tone === 'down'
        ? theme.semantic.bg.dangerSoft
        : theme.semantic.bg.canvas};
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

export const SideCard = styled.div<{ $side: 'up' | 'down'; $winner?: boolean }>`
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
  padding: 14px;
  border-radius: 10px;
  border: 1px solid
    ${({ theme, $side, $winner }) =>
      $winner
        ? $side === 'up'
          ? theme.semantic.status.success
          : theme.semantic.border.danger
        : theme.semantic.border.default};
  background: ${({ theme }) => theme.semantic.bg.canvas};
`;

export const SideHead = styled.div`
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
`;

export const ScoreBlock = styled.div`
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
`;

export const ScoreNum = styled.span`
  font-size: 28px;
  font-weight: 800;
  line-height: 1;
  letter-spacing: -0.03em;
  color: ${({ theme }) => theme.semantic.text.primary};
`;

export const EaseChip = styled.span<{ $tone: 'easier' | 'likely' | 'mixed' | 'hard' }>`
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.02em;
  text-transform: uppercase;
  color: ${({ theme, $tone }) =>
    $tone === 'easier' || $tone === 'likely'
      ? theme.semantic.text.primary
      : theme.semantic.text.secondary};
  background: ${({ theme, $tone }) =>
    $tone === 'easier'
      ? theme.semantic.bg.successSoft
      : $tone === 'likely'
        ? theme.semantic.bg.accentSoft
        : theme.semantic.bg.accentMuted};
`;

export const ScoreBar = styled.div`
  height: 6px;
  border-radius: 99px;
  background: ${({ theme }) => theme.semantic.bg.accentMuted};
  overflow: hidden;
`;

export const ScoreFill = styled.div<{ $side: 'up' | 'down'; $pct: number }>`
  height: 100%;
  width: ${({ $pct }) => `${Math.max(0, Math.min(100, $pct))}%`};
  background: ${({ theme, $side }) =>
    $side === 'up' ? theme.semantic.status.success : theme.semantic.status.error};
`;

export const MetricTable = styled.dl`
  display: grid;
  grid-template-columns: minmax(92px, 34%) 1fr;
  gap: 6px 10px;
  margin: 0;
`;

export const MetricLabel = styled.dt`
  margin: 0;
  font-size: 12px;
  color: ${({ theme }) => theme.semantic.text.secondary};
`;

export const MetricValue = styled.dd<{ $tone?: 'up' | 'down' | 'muted' | 'profit' | 'loss' }>`
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: ${({ theme, $tone }) =>
    $tone === 'up' || $tone === 'profit'
      ? theme.semantic.status.success
      : $tone === 'down' || $tone === 'loss'
        ? theme.semantic.status.error
        : theme.semantic.text.primary};
`;

export const ReasonList = styled.ul`
  margin: 0;
  padding-left: 18px;
  display: flex;
  flex-direction: column;
  gap: 4px;
`;

export const ReasonItem = styled.li`
  font-size: 12px;
  line-height: 1.4;
  color: ${({ theme }) => theme.semantic.text.secondary};
`;

export const Hint = styled.p`
  margin: 0;
  font-size: 12px;
  line-height: 1.45;
  color: ${({ theme }) => theme.semantic.text.secondary};
`;

export const CoverageRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
`;

export const CoverageChip = styled.span<{ $tone: 'complete' | 'usable' | 'thin' | 'insufficient' }>`
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.02em;
  text-transform: uppercase;
  color: ${({ theme }) => theme.semantic.text.primary};
  background: ${({ theme, $tone }) =>
    $tone === 'complete'
      ? theme.semantic.bg.successSoft
      : $tone === 'usable'
        ? theme.semantic.bg.accentSoft
        : $tone === 'thin'
          ? theme.semantic.bg.accentMuted
          : theme.semantic.bg.dangerSoft};
`;

export const CoverageMeter = styled.div`
  flex: 1 1 120px;
  min-width: 80px;
  max-width: 220px;
  height: 6px;
  border-radius: 99px;
  background: ${({ theme }) => theme.semantic.bg.accentMuted};
  overflow: hidden;
`;

export const CoverageFill = styled.div<{ $pct: number; $tone: 'complete' | 'usable' | 'thin' | 'insufficient' }>`
  height: 100%;
  width: ${({ $pct }) => `${Math.max(0, Math.min(100, $pct))}%`};
  background: ${({ theme, $tone }) =>
    $tone === 'complete' || $tone === 'usable'
      ? theme.semantic.status.success
      : $tone === 'thin'
        ? theme.semantic.status.warning
        : theme.semantic.status.error};
`;

export const InputList = styled.ul`
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
`;

export const InputPill = styled.li<{ $tone: 'ok' | 'weak' | 'missing' | 'error' }>`
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 99px;
  font-size: 11px;
  font-weight: 600;
  color: ${({ theme, $tone }) =>
    $tone === 'ok'
      ? theme.semantic.status.success
      : $tone === 'weak'
        ? theme.semantic.status.warning
        : theme.semantic.status.error};
  background: ${({ theme, $tone }) =>
    $tone === 'ok'
      ? theme.semantic.bg.successSoft
      : $tone === 'error' || $tone === 'missing'
        ? theme.semantic.bg.dangerSoft
        : theme.semantic.bg.accentMuted};
`;
