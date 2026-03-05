interface StatusBannerProps {
  loading: boolean;
  error: string | null;
  onRetry: () => void;
}

export default function StatusBanner({ loading, error, onRetry }: StatusBannerProps) {
  if (loading) {
    return <div className="status-banner loading">Loading system metrics...</div>;
  }

  if (error) {
    return (
      <div className="status-banner error">
        <span>{error}</span>
        <button onClick={onRetry} type="button">
          Retry
        </button>
      </div>
    );
  }

  return <div className="status-banner success">Connected</div>;
}
