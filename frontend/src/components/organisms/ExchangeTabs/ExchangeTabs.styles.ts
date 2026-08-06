import styled from 'styled-components';
import { Tabs } from 'antd';

export const StyledTabs = styled(Tabs)`
  .ant-tabs-nav {
    margin-bottom: ${({ theme }) => theme.spacing[4]}px;
  }

  .ant-tabs-tab {
    color: ${({ theme }) => theme.semantic.text.secondary};
  }

  .ant-tabs-tab-active .ant-tabs-tab-btn {
    color: ${({ theme }) => theme.semantic.text.primary} !important;
    font-weight: 600;
  }

  .ant-tabs-ink-bar {
    background: ${({ theme }) => theme.semantic.accent.default};
  }
`;
