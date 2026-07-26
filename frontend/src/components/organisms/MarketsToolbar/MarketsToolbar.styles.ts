import styled from 'styled-components';
import { Select } from 'antd';

export const ToolbarRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[3]}px;
  align-items: center;
  margin-bottom: ${({ theme }) => theme.spacing[4]}px;
`;

export const SearchWrap = styled.div`
  flex: 1 1 220px;
  min-width: 180px;
  max-width: 360px;
`;

export const FieldWrap = styled.div`
  flex: 0 0 auto;
`;

export const QuoteSelect = styled(Select)`
  width: 120px;
` as typeof Select;

export const TagSelect = styled(Select)`
  min-width: 160px;
` as typeof Select;
