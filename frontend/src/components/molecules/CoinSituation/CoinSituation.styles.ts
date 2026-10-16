import { Link } from 'react-router-dom';
import styled from 'styled-components';

export const Card = styled.section`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[1]}px;
  padding: ${({ theme }) => theme.spacing[3]}px;
  border: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  border-radius: ${({ theme }) => theme.radii.md}px;
  background: ${({ theme }) => theme.semantic.bg.elevated};
`;

export const CardLink = styled(Link)`
  align-self: flex-start;
  color: ${({ theme }) => theme.semantic.text.link};
  font-size: ${({ theme }) => theme.typeScale.caption.fontSize}px;
  text-decoration: underline;
  text-underline-offset: 2px;
`;
