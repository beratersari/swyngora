import styled from 'styled-components';

export const Card = styled.section`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;
  padding: ${({ theme }) => theme.spacing[4]}px;
  background: ${({ theme }) => theme.semantic.bg.surface};
  border: 1px solid ${({ theme }) => theme.semantic.border.default};
  border-radius: ${({ theme }) => theme.radii.md}px;
`;

export const List = styled.ol`
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
`;

export const Item = styled.li`
  display: grid;
  grid-template-columns: 16px minmax(0, 1fr);
  column-gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;
`;

export const Rail = styled.span`
  position: relative;
  display: flex;
  justify-content: center;
  min-height: 100%;
  &::before {
    content: '';
    position: absolute;
    top: 8px;
    bottom: -8px;
    left: 50%;
    width: 1px;
    background: ${({ theme }) => theme.semantic.border.default};
    transform: translateX(-50%);
  }
  ${Item}:last-child &::before {
    display: none;
  }
`;

export const Dot = styled.span`
  position: relative;
  z-index: 1;
  width: 10px;
  height: 10px;
  margin-top: 5px;
  border-radius: 50%;
  background: ${({ theme }) => theme.semantic.accent.default};
`;

export const Body = styled.div`
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  padding-bottom: ${({ theme }) => theme.spacing[3]}px;
  ${Item}:last-child & {
    padding-bottom: 0;
  }
`;

export const Meta = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: baseline;
`;
