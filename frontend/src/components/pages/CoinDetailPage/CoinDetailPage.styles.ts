import styled from 'styled-components';
import { Tabs } from 'antd';

export const PageStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[4]}px;
  width: 100%;
  min-width: 0;
`;

export const ChartCard = styled.section`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
  padding: ${({ theme }) => theme.spacing[3]}px;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  border: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  border-radius: ${({ theme }) => theme.radii.md}px;
`;

export const ChartTitleRow = styled.div`
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  justify-content: space-between;
  gap: ${({ theme }) => theme.spacing[2]}px;
`;

export const ChartAndBook = styled.div`
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: ${({ theme }) => theme.spacing[4]}px;
  align-items: start;

  @media (min-width: 1100px) {
    grid-template-columns: minmax(0, 1fr) minmax(300px, 380px);
  }
`;

export const SideStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;
`;

export const DeskTabs = styled(Tabs)`
  min-width: 0;
  max-width: 100%;

  .ant-tabs-nav {
    margin-bottom: ${({ theme }) => theme.spacing[3]}px;
  }

  .ant-tabs-nav::before {
    border-bottom-color: ${({ theme }) => theme.semantic.border.subtle};
  }

  .ant-tabs-tab {
    padding: 8px 14px;
    font-size: 13px;
  }

  .ant-tabs-content-holder,
  .ant-tabs-tabpane {
    min-width: 0;
    max-width: 100%;
  }
`;

export const TabStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[4]}px;
  min-width: 0;
`;

export const PaperTradeCard = styled.section`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[2]}px;
  padding: ${({ theme }) => theme.spacing[3]}px;
  background: ${({ theme }) => theme.semantic.bg.canvas};
  border: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  border-radius: ${({ theme }) => theme.radii.md}px;
`;
