import styled from 'styled-components';

export const Grid = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(280px, 100%), 1fr));
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
  min-height: 168px;
  min-width: 0;
  overflow: hidden;
  transition:
    transform ${({ theme }) => theme.motion.duration.base} ${({ theme }) => theme.motion.ease.standard},
    border-color ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard},
    box-shadow ${({ theme }) => theme.motion.duration.base} ${({ theme }) => theme.motion.ease.standard};

  &:hover {
    transform: translateY(-2px);
    border-color: ${({ theme }) => theme.semantic.border.accent};
    box-shadow: 0 8px 24px rgba(0, 15, 15, 0.35);
  }

  @media (prefers-reduced-motion: reduce) {
    &:hover {
      transform: none;
    }
  }
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

export const SummaryList = styled.ul`
  margin: 0;
  padding-left: 1.1em;
  display: flex;
  flex-direction: column;
  gap: 2px;
`;

export const CardFooter = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[2]}px;
  margin-top: auto;
`;
