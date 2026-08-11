import styled from 'styled-components';
import { Link } from 'react-router-dom';

export const HeaderCard = styled.section`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
  padding: ${({ theme }) => theme.spacing[2]}px 0 ${({ theme }) => theme.spacing[1]}px;
`;

export const TopRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

export const TitleBlock = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[1]}px;
`;

export const TitleRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

export const PriceBlock = styled.div`
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: ${({ theme }) => theme.spacing[1]}px;
  text-align: right;

  ${({ theme }) => theme.media.phone} {
    align-items: flex-start;
    text-align: left;
    width: 100%;
  }
`;

export const BackLink = styled(Link)`
  color: ${({ theme }) => theme.semantic.text.secondary};
  text-decoration: none;
  font-size: 13px;

  &:hover {
    color: ${({ theme }) => theme.semantic.text.primary};
  }
`;
