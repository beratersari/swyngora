import { Component, type ErrorInfo, type ReactNode } from 'react';
import { Button, Result } from 'antd';
import { i18n } from '@/libs/i18n';
import { ErrorBoundaryShell } from './ErrorBoundary.styles';

type Props = {
  children: ReactNode;
  /** Optional test hook / override */
  fallbackTitle?: string;
  fallbackBody?: string;
};

type State = {
  hasError: boolean;
  message?: string;
};

/**
 * Catches render errors so a white screen is not the only outcome.
 * Recovery: full reload (resets Redux + router cleanly).
 */
export class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false };

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, message: error?.message };
  }

  componentDidCatch(error: Error, info: ErrorInfo): void {
    if (import.meta.env.DEV) {
      // eslint-disable-next-line no-console
      console.error('[ErrorBoundary]', error, info.componentStack);
    }
  }

  private handleReload = () => {
    window.location.assign(window.location.pathname + window.location.search);
  };

  private handleReset = () => {
    this.setState({ hasError: false, message: undefined });
  };

  render() {
    if (!this.state.hasError) {
      return this.props.children;
    }

    const title =
      this.props.fallbackTitle ?? i18n.t('common:errors.boundaryTitle');
    const body =
      this.props.fallbackBody ?? i18n.t('common:errors.boundaryBody');

    return (
      <ErrorBoundaryShell role="alert">
        <Result
          status="error"
          title={title}
          subTitle={body}
          extra={[
            <Button key="retry" type="primary" onClick={this.handleReset}>
              {i18n.t('common:actions.retry')}
            </Button>,
            <Button key="reload" onClick={this.handleReload}>
              {i18n.t('common:actions.reload')}
            </Button>,
          ]}
        />
      </ErrorBoundaryShell>
    );
  }
}
