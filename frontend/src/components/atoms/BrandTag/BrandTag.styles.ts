import styled, { css } from 'styled-components';
import { Tag } from 'antd';
import type { BrandTagVariant } from './BrandTag.types';

const variants: Record<BrandTagVariant, ReturnType<typeof css>> = {
  status: css`
    background: rgba(3, 98, 76, 0.45) !important;
    border: 1px solid ${({ theme }) => theme.palette.frog} !important;
    color: ${({ theme }) => theme.palette.mint} !important;
  `,
  live: css`
    background: rgba(0, 255, 129, 0.12) !important;
    border: 1px solid ${({ theme }) => theme.palette.caribbeanGreen} !important;
    color: ${({ theme }) => theme.palette.caribbeanGreen} !important;
  `,
  exchange: css`
    background: ${({ theme }) => theme.palette.pine} !important;
    border: 1px solid ${({ theme }) => theme.semantic.border.default} !important;
    color: ${({ theme }) => theme.palette.pistachio} !important;
  `,
  paused: css`
    background: rgba(112, 125, 125, 0.2) !important;
    border: 1px solid ${({ theme }) => theme.semantic.border.default} !important;
    color: ${({ theme }) => theme.semantic.text.secondary} !important;
  `,
  delist: css`
    background: rgba(224, 184, 106, 0.16) !important;
    border: 1px solid ${({ theme }) => theme.semantic.status.warning} !important;
    color: ${({ theme }) => theme.semantic.status.warning} !important;
  `,
};

export const StyledBrandTag = styled(Tag)<{ $variant: BrandTagVariant }>`
  && {
    margin-inline-end: 0;
    line-height: 1.4;
    font-weight: 500;
    ${({ $variant }) => variants[$variant]}
  }
`;
