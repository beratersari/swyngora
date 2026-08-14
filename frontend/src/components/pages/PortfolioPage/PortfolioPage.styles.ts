import styled from 'styled-components';
import { Tabs } from 'antd';

export const PageStack = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[5]}px;
  width: 100%;
  min-width: 0;
  max-width: 100%;

  ${({ theme }) => theme.media.tablet} {
    gap: ${({ theme }) => theme.spacing[4]}px;
  }

  ${({ theme }) => theme.media.phone} {
    gap: ${({ theme }) => theme.spacing[3]}px;
  }
`;

export const Section = styled.section`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  min-width: 0;
  max-width: 100%;
`;

/** Trade + cash panels: two columns on desk, one column from tablet down. */
export const Split = styled.div`
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(0, 1fr);
  gap: ${({ theme }) => theme.spacing[4]}px;
  min-width: 0;
  max-width: 100%;

  ${({ theme }) => theme.media.tablet} {
    grid-template-columns: 1fr;
    gap: ${({ theme }) => theme.spacing[3]}px;
  }
`;

export const PanelCard = styled.div`
  display: flex;
  flex-direction: column;
  gap: ${({ theme }) => theme.spacing[3]}px;
  padding: ${({ theme }) => theme.spacing[4]}px;
  border-radius: ${({ theme }) => theme.radii.md}px;
  border: 1px solid ${({ theme }) => theme.semantic.border.subtle};
  background: ${({ theme }) => theme.semantic.bg.elevated};
  min-width: 0;
  max-width: 100%;
  overflow: hidden;

  ${({ theme }) => theme.media.phone} {
    padding: ${({ theme }) => theme.spacing[3]}px;
    gap: ${({ theme }) => theme.spacing[2]}px;
  }
`;

/** Keep Ant Tabs from blowing page width on narrow viewports. */
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
    padding: 8px 12px;
    font-size: 13px;
  }

  .ant-tabs-content-holder {
    min-width: 0;
    max-width: 100%;
  }

  .ant-tabs-tabpane {
    min-width: 0;
    max-width: 100%;
  }

  ${({ theme }) => theme.media.phone} {
    .ant-tabs-nav {
      margin-bottom: ${({ theme }) => theme.spacing[2]}px;
    }

    .ant-tabs-tab {
      padding: 6px 10px;
      font-size: 12px;
    }

    .ant-tabs-nav-list {
      gap: 0;
    }
  }
`;
