// totalSeries sums a metric's head series over its sources, or undefined
// when no source recorded it.
export function totalSeries(
  series?: Record<string, number>,
): number | undefined {
  if (!series) return undefined;
  const values = Object.values(series);
  return values.length ? values.reduce((a, b) => a + b, 0) : undefined;
}
