// totalSeries sums a metric's head series over its sources, or undefined
// when no source recorded it.
export function totalSeries(
  series?: Record<string, number>,
): number | undefined {
  if (!series) return undefined;
  const values = Object.values(series);
  return values.length ? values.reduce((a, b) => a + b, 0) : undefined;
}

// UPSTREAM_SERIES_SOURCE is the source a metric's series come under when no
// metadata source counted it: its series summed over jobs, counted in the
// upstream by the job index (db.UpstreamSeriesSource).
export const UPSTREAM_SERIES_SOURCE = "upstream";

// seriesBreakdown is the per-source tooltip for a metric's series.
export function seriesBreakdown(series?: Record<string, number>): string {
  return Object.entries(series ?? {})
    .map(([source, n]) =>
      source === UPSTREAM_SERIES_SOURCE
        ? `${n} in upstream, all jobs (no source's top 10000)`
        : `${source}: ${n}`,
    )
    .join("\n");
}
