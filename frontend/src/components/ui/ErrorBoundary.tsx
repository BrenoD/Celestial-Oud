import { Component, type ErrorInfo, type ReactNode } from 'react';

interface State {
  error: Error | null;
}

export class ErrorBoundary extends Component<{ children: ReactNode }, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    // Hook for Sentry/OpenTelemetry later; keep console for now.
    console.error('Unhandled UI error', error, info.componentStack);
  }

  render() {
    if (!this.state.error) return this.props.children;
    return (
      <main className="container page" style={{ textAlign: 'center', paddingTop: 80 }}>
        <h1 className="page-title">Something went wrong</h1>
        <p className="page-intro">Please refresh the page. If it keeps happening, contact us and we will sort it out.</p>
        <p style={{ marginTop: 24 }}>
          <a href="/" className="btn btn--primary">
            Back to home
          </a>
        </p>
      </main>
    );
  }
}
