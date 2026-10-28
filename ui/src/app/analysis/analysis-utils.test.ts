import { describe, it, expect } from "vitest";
import {
  catalogLink,
  groupLabel,
  totals,
  unusedPercent,
} from "./analysis-utils";

const row = { job: "api", source: "source-a", total: 8, unused: 3 };

describe("analysis utils", () => {
  it("labels a row by its grouping", () => {
    expect(groupLabel(row, "job")).toBe("api");
    expect(groupLabel(row, "source")).toBe("source-a");
    expect(groupLabel(row, "job,source")).toBe("api @ source-a");
    expect(groupLabel({ total: 1, unused: 0 }, "job")).toBe("—");
  });

  it("rounds the unused share and survives an empty group", () => {
    expect(unusedPercent(row)).toBe(38);
    expect(unusedPercent({ total: 0, unused: 0 })).toBe(0);
  });

  it("links a group to its unused metrics in the catalog", () => {
    expect(catalogLink(row)).toBe(
      "/metrics-explorer?usage=unused&job=api&source=source-a",
    );
    expect(catalogLink({ job: "api", total: 1, unused: 1 })).toBe(
      "/metrics-explorer?usage=unused&job=api",
    );
  });

  it("sums totals", () => {
    expect(totals([row, { total: 2, unused: 2 }])).toEqual({
      total: 10,
      unused: 5,
    });
  });
});
