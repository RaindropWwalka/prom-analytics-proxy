import type { Producer } from "@/lib/types";

// ProducerFilters narrows a metric's producers: text filters match
// case-insensitively, minSeries keeps producers at or above it. Empty means
// no filter.
export interface ProducerFilters {
  job: string;
  service: string;
  source: string;
  minSeries: string;
}

export const NO_PRODUCER_FILTERS: ProducerFilters = {
  job: "",
  service: "",
  source: "",
  minSeries: "",
};

export function filterProducers(
  producers: Producer[],
  f: ProducerFilters,
): Producer[] {
  const text = (value: string | undefined, needle: string) =>
    needle.trim() === "" ||
    (value ?? "").toLowerCase().includes(needle.trim().toLowerCase());
  const min = Number(f.minSeries);
  return producers.filter(
    (p) =>
      text(p.job, f.job) &&
      text(p.service, f.service) &&
      text(p.source, f.source) &&
      (f.minSeries.trim() === "" || Number.isNaN(min) || p.series >= min),
  );
}
