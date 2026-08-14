import styled from 'styled-components';

export { FormStack, FieldRow, Field, Actions } from '../PaperTradeForm/PaperTradeForm.styles';

export const ModeRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[2]}px;
  align-items: center;
`;
