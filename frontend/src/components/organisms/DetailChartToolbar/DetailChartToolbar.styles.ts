import styled from 'styled-components';

export const ToolbarRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

export const Field = styled.label<{ $compact?: boolean }>`
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: ${({ $compact }) => ($compact ? '88px' : '100px')};
  max-width: 100%;

  ${({ theme }) => theme.media.phone} {
    min-width: 0;
    flex: 1 1 100%;
  }
`;

/** Label + small switch on one row (keeps the toggle compact). */
export const InlineField = styled.label`
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 24px;
  padding-bottom: 2px;
`;
