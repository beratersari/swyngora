import styled from 'styled-components';

export const PageStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;
  width: 100%;
`;

export const Toolbar = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const Field = styled.label`
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 8px;

  > span {
    font-size: 12px;
    font-weight: 600;
    color: ${({ theme }) => theme.semantic.text.secondary};
  }
`;

export const OverviewLayout = styled.div`
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(280px, 340px);
  gap: ${({ theme }) => theme.spacing[3]}px;
  align-items: stretch;
  min-height: min(620px, calc(100dvh - 280px));

  @media (max-width: 960px) {
    grid-template-columns: 1fr;
    min-height: 0;
  }
`;

export const MapCol = styled.div`
  min-width: 0;
  min-height: 460px;
  display: flex;
  flex-direction: column;
`;

export const CardsCol = styled.div`
  min-width: 0;
`;

export const HeatmapStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;
`;
