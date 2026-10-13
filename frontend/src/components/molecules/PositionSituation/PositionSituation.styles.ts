import styled from 'styled-components';

export const Stack = styled.div`
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-width: 28rem;
`;

export const Lot = styled.div`
  display: flex;
  flex-direction: column;
  gap: 2px;

  & + & {
    margin-top: 8px;
  }
`;
