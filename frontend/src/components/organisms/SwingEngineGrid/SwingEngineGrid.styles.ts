import styled from 'styled-components';

export const Grid = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(300px, 100%), 1fr));
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

export const Card = styled.article`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
  padding: ${({ theme }) => theme.spacing[4]}px;
  background: ${({ theme }) => theme.semantic.bg.muted};
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.md}px;
  min-height: 188px;
  min-width: 0;
  overflow: hidden;
`;

export const CardTop = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const PairBlock = styled.div`
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
`;

export const FactorRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
`;

export const Levels = styled.div`
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
`;

export const LevelRow = styled.div`
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;
`;

export const LevelLabel = styled.span`
  flex: 0 0 auto;
  color: ${({ theme }) => theme.semantic.text.secondary};
  font-size: 12px;
`;

export const LevelValue = styled.span`
  flex: 1 1 auto;
  min-width: 0;
  text-align: right;
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-variant-numeric: tabular-nums;
  font-size: 13px;
  font-weight: 600;
  color: ${({ theme }) => theme.semantic.text.primary};
  overflow-wrap: anywhere;
`;

export const CardFooter = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[2]}px;
  margin-top: auto;
`;
