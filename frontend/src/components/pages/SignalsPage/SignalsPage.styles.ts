import styled from 'styled-components';

export const PageStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[5]}px;
`;

export const Section = styled.section`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

export const MetricStrip = styled.div`
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

export const MetricCard = styled.div`
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: ${({ theme }) => theme.spacing[3]}px ${({ theme }) => theme.spacing[4]}px;
  background: ${({ theme }) => theme.semantic.bg.muted};
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.md}px;
  transition:
    border-color ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard},
    transform ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard};

  &:hover {
    border-color: ${({ theme }) => theme.semantic.border.accent};
    transform: translateY(-1px);
  }

  @media (prefers-reduced-motion: reduce) {
    &:hover {
      transform: none;
    }
  }
`;

export const DeskSplit = styled.div`
  display: grid;
  grid-template-columns: minmax(280px, 1fr) minmax(320px, 1.1fr);
  gap: ${({ theme }) => theme.spacing[5]}px;

  @media (max-width: 960px) {
    grid-template-columns: 1fr;
  }
`;
