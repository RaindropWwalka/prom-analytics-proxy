import { describe, it, expect } from "vitest";
import { filterProducers, NO_PRODUCER_FILTERS } from "./producer-filters";

const producers = [
  { job: "jmx", service: "Billing", source: "source-a", series: 120 },
  { job: "node", source: "source-b", series: 4 },
];

describe("filterProducers", () => {
  it("keeps everything without filters", () => {
    expect(filterProducers(producers, NO_PRODUCER_FILTERS)).toHaveLength(2);
  });

  it("matches text case-insensitively and series as a floor", () => {
    expect(
      filterProducers(producers, { ...NO_PRODUCER_FILTERS, service: "bill" }),
    ).toEqual([producers[0]]);
    expect(
      filterProducers(producers, { ...NO_PRODUCER_FILTERS, source: "B" }),
    ).toEqual([producers[1]]);
    expect(
      filterProducers(producers, { ...NO_PRODUCER_FILTERS, minSeries: "10" }),
    ).toEqual([producers[0]]);
  });
});
