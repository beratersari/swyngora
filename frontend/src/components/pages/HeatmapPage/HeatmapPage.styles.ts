import styled from 'styled-components';

export const PageStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: 0;
  min-width: 0;
  width: 100%;
  height: 100%;
  min-height: 0;
`;

export const Chrome = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[2]}px;
  min-height: 44px;
  padding: 6px ${({ theme }) => theme.spacing[3]}px;
  border-bottom: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  background: ${({ theme }) => theme.semantic.bg.page};
`;

export const ChromeLeft = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

export const Field = styled.label`
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 8px;

  > span {
    font-size: 10px;
    font-weight: 600;
    letter-spacing: 0.1em;
    text-transform: uppercase;
    color: ${({ theme }) => theme.semantic.text.tertiary};
  }
`;

export const BoardWrap = styled.div`
  flex: 1 1 auto;
  min-height: 0;
  height: calc(100dvh - 168px);
  display: flex;
  flex-direction: column;
`;

export const Title = styled.h1`
  margin: 0;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: ${({ theme }) => theme.semantic.text.tertiary};
`;

export const Legend = styled.div`
  display: none;
`;

export const LegendBar = styled.div``;

export const LegendTick = styled.span``;
