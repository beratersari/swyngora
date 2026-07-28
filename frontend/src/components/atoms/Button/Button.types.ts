import type { ButtonProps as AntButtonProps } from 'antd';
import type { WithLoadingProps } from '@/components/types';

export type ButtonProps = Omit<AntButtonProps, 'loading'> &
  WithLoadingProps & {
    /**
     * When true, renders a button-shaped Skeleton instead of the control.
     * For in-button spinner, use `pending` (disables unless `disabled` is set).
     */
    isLoading?: boolean;
    /** Maps to Ant Design `loading` spinner; disables when `disabled` is omitted */
    pending?: boolean;
  };
