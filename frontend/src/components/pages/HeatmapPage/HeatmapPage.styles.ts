import styled from 'styled-components';

export const PageStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;
  width: 100%;
`;

export const Toolbar = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const ToolbarLeft = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[3]}px;
`;

export const ToolbarRight = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const Field = styled.label`
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 8px;

  > span {
    font-size: 12px;
    font-weight: 600;
    color: ${({ theme }) => theme.semantic.text.secondary};
  }
`;

export const BoardWrap = styled.div`
  flex: 1 1 auto;
  min-height: 0;
  height: min(720px, calc(100dvh - 280px));
  min-height: 460px;
  display: flex;
  flex-direction: column;

  &:fullscreen {
    height: 100%;
    min-height: 100%;
    padding: 16px;
    background: ${({ theme }) => theme.semantic.bg.page};
    box-sizing: border-box;
  }
`;
