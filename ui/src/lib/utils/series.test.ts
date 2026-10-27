import { describe, it, expect } from "vitest";
import { totalSeries } from "./series";

describe("totalSeries", () => {
  it("sums series over sources", () => {
    expect(totalSeries({ "source-a": 12, "source-b": 3 })).toBe(15);
  });

  it("is undefined when no source recorded the metric", () => {
    expect(totalSeries(undefined)).toBeUndefined();
    expect(totalSeries({})).toBeUndefined();
  });
});
