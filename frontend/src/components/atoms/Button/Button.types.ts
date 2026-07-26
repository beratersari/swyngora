import type { ButtonProps as AntButtonProps } from 'antd';
import type { WithLoadingProps } from '@/components/types';

export type ButtonProps = Omit<AntButtonProps, 'loading'> &
  WithLoadingProps & {
    /**
     * When true, renders a button-shaped Skeleton instead of the control.
     * For in-button spinner (still interactive chrome), use `pending` instead.
     */
    isLoading?: boolean;
    /** Maps to Ant Design `loading` spinner on the button */
    pending?: boolean;
  };
