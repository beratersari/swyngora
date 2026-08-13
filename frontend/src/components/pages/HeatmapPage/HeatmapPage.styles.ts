import styled from 'styled-components';

export const PageStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;
  width: 100%;
`;

export const Chrome = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-height: 40px;
`;

export const ChromeLeft = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

export const Field = styled.label`
  display: flex;
  flex-direction: column;
  gap: 4px;
`;

export const Legend = styled.div`
  display: flex;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[2]}px;
  padding-bottom: 4px;
`;

export const LegendBar = styled.div`
  width: 168px;
  height: 5px;
  border-radius: 1px;
`;

export const LegendTick = styled.span`
  font-family: ${({ theme }) => theme.fontFamilies.mono};
  font-variant-numeric: tabular-nums;
  font-size: ${({ theme }) => theme.typeScale.overline.fontSize}px;
  color: ${({ theme }) => theme.semantic.text.tertiary};
`;
