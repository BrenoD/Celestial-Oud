interface Props {
  rating: number;
  count?: number;
  size?: number;
  showCount?: boolean;
}

export function Stars({ rating, count, size = 13, showCount = true }: Props) {
  const rounded = Math.round(rating);
  return (
    <span style={{ fontSize: size, color: 'var(--muted-2)' }}>
      <span className="stars" aria-label={`Rated ${rating.toFixed(1)} out of 5`} role="img">
        {'★'.repeat(rounded)}
        <span style={{ color: 'var(--line-3)' }}>{'★'.repeat(5 - rounded)}</span>
      </span>
      {showCount && count !== undefined && <> ({count})</>}
    </span>
  );
}
