import { Component, type ErrorInfo, type ReactNode } from "react";

type Props = {
  children: ReactNode;
};

type State = {
  failed: boolean;
};

export class RouteErrorBoundary extends Component<Props, State> {
  state: State = { failed: false };

  static getDerivedStateFromError(): State {
    return { failed: true };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("Admin route render failed", error, info);
  }

  render() {
    if (this.state.failed) {
      return <section className="panel route-error">
        <span className="kicker">EKRAN KURTARILDI</span>
        <h2>Bu ekranın verileri görüntülenemedi</h2>
        <p>Sayfa tamamen kapanmadı. Verileri yeniden yükleyerek güvenle tekrar deneyebilirsiniz.</p>
        <button className="primary-button" onClick={() => window.location.reload()}>Verileri yeniden yükle</button>
      </section>;
    }
    return this.props.children;
  }
}
