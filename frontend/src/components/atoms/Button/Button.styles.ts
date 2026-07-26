import styled from 'styled-components';
import { Button as AntButton } from 'antd';

/** Styled Ant Design Button — brand-aligned via ConfigProvider + local tweaks */
export const StyledButton = styled(AntButton)`
  font-family: ${({ theme }) => theme.fontFamilies.sans};
`;
