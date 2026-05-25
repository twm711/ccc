import { Component } from 'react';
import type { ReactNode, ErrorInfo } from 'react';
import { Alert, Button } from 'antd';

interface Props {
  children: ReactNode;
}

interface State {
  hasError: boolean;
  error: Error | null;
}

export default class ErrorBoundary extends Component<Props, State> {
  state: State = { hasError: false, error: null };

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('[ErrorBoundary]', error, info.componentStack);
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null });
  };

  render() {
    if (this.state.hasError) {
      return (
        <div style={{ padding: 40 }}>
          <Alert
            type="error"
            message="页面出现异常"
            description={this.state.error?.message || '未知错误'}
            action={<Button onClick={this.handleReset}>重试</Button>}
          />
        </div>
      );
    }
    return this.props.children;
  }
}
