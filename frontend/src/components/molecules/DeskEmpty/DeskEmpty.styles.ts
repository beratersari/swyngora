import styled from 'styled-components';

export const Wrap = styled.div`
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: ${({ theme }) => theme.spacing[2]}px;
  padding: ${({ theme }) => theme.spacing[8]}px ${({ theme }) => theme.spacing[4]}px;
  text-align: center;
`;

export const Mark = styled.div`
  width: 36px;
  height: 36px;
  border-radius: ${({ theme }) => theme.radii.md}px;
  border: 1px dashed ${({ theme }) => theme.semantic.border.default};
  background: ${({ theme }) => theme.semantic.bg.chrome};
  margin-bottom: ${({ theme }) => theme.spacing[1]}px;
`;
