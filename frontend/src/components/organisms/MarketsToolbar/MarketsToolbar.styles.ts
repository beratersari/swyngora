import styled from 'styled-components';
import { Select } from 'antd';

export const ToolbarRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  gap: ${({ theme }) => theme.spacing[3]}px;
  align-items: center;
  margin-bottom: ${({ theme }) => theme.spacing[4]}px;

  ${({ theme }) => theme.media.phone} {
    gap: ${({ theme }) => theme.spacing[2]}px;
    margin-bottom: ${({ theme }) => theme.spacing[2]}px;
  }
`;

export const SearchWrap = styled.div`
  flex: 1 1 220px;
  min-width: 180px;
  max-width: 360px;

  ${({ theme }) => theme.media.phone} {
    flex: 1 1 100%;
    min-width: 0;
    max-width: none;
  }
`;

export const FieldWrap = styled.div`
  flex: 0 0 auto;

  ${({ theme }) => theme.media.phone} {
    flex: 1 1 calc(50% - ${({ theme }) => theme.spacing[2]}px);
    min-width: 0;
  }
`;

export const QuoteSelect = styled(Select)`
  width: 120px;

  ${({ theme }) => theme.media.phone} {
    width: 100%;
  }
` as typeof Select;

export const TagSelect = styled(Select)`
  min-width: 160px;

  ${({ theme }) => theme.media.phone} {
    width: 100%;
    min-width: 0;
  }
` as typeof Select;
