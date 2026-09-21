import styled from 'styled-components';

export const Track = styled.span`
  display: flex;
  width: 100%;
  height: 10px;
  overflow: hidden;
  border-radius: ${({ theme }) => theme.radii.pill}px;
  background: ${({ theme }) => theme.semantic.bg.accentMuted};
`;

export const Fill = styled.span<{ $color: string; $width: string }>`
  display: block;
  width: ${({ $width }) => $width};
  height: 100%;
  background: ${({ $color }) => $color};
`;
