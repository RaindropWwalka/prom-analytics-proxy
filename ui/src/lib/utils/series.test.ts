import { describe, it, expect } from "vitest";
import { seriesBreakdown, totalSeries } from "./series";

describe("totalSeries", () => {
  it("sums series over sources", () => {
    expect(totalSeries({ "source-a": 12, "source-b": 3 })).toBe(15);
  });

  it("is undefined when no source recorded the metric", () => {
    expect(totalSeries(undefined)).toBeUndefined();
    expect(totalSeries({})).toBeUndefined();
  });
});

describe("seriesBreakdown", () => {
  it("lists each source's series", () => {
    expect(seriesBreakdown({ "source-a": 12, "source-b": 3 })).toBe(
      "source-a: 12\nsource-b: 3",
    );
  });

  it("says where an upstream count comes from", () => {
    expect(seriesBreakdown({ upstream: 42 })).toBe(
      "42 in upstream, all jobs (no source's top 10000)",
    );
  });
});
