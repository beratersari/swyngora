import styled, { css } from 'styled-components';
import { Tag } from 'antd';
import type { BrandTagVariant } from './BrandTag.types';

const variants: Record<BrandTagVariant, ReturnType<typeof css>> = {
  status: css`
    background: ${({ theme }) => theme.palette.bangladeshGreen} !important;
    border: 1px solid ${({ theme }) => theme.palette.frog} !important;
    color: ${({ theme }) => theme.palette.antiFlashWhite} !important;
  `,
  live: css`
    background: ${({ theme }) => theme.semantic.bg.accentSoft} !important;
    border: 1px solid ${({ theme }) => theme.palette.frog} !important;
    color: ${({ theme }) => theme.palette.mint} !important;
  `,
  exchange: css`
    background: ${({ theme }) => theme.palette.bangladeshGreen} !important;
    border: 1px solid ${({ theme }) => theme.palette.frog} !important;
    color: ${({ theme }) => theme.palette.antiFlashWhite} !important;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    font-size: 11px;
    font-weight: 700;
  `,
  paused: css`
    background: ${({ theme }) => theme.semantic.bg.chrome} !important;
    border: 1px solid ${({ theme }) => theme.semantic.border.default} !important;
    color: ${({ theme }) => theme.semantic.text.primary} !important;
  `,
  delist: css`
    background: rgba(224, 184, 106, 0.18) !important;
    border: 1px solid ${({ theme }) => theme.semantic.status.warning} !important;
    color: ${({ theme }) => theme.palette.antiFlashWhite} !important;
  `,
};

export const StyledBrandTag = styled(Tag)<{ $variant: BrandTagVariant }>`
  && {
    margin-inline-end: 0;
    line-height: 1.4;
    font-weight: 500;
    transition:
      background ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard},
      border-color ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard},
      color ${({ theme }) => theme.motion.duration.fast} ${({ theme }) => theme.motion.ease.standard};
    ${({ $variant }) => variants[$variant]}
  }
`;
