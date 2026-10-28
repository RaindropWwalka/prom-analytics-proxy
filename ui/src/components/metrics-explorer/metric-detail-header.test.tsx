import { render, screen } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { MetricDetailHeader } from "./metric-detail-header";

const useMetricMetadata = vi.fn();
vi.mock("@/app/metrics/use-metrics-data", () => ({
  useMetricMetadata: (name: string) => useMetricMetadata(name),
}));
vi.mock("@/components/preserved-link.tsx", () => ({
  PreservedLink: ({ children }: { children: React.ReactNode }) => (
    <a>{children}</a>
  ),
}));

describe("MetricDetailHeader", () => {
  beforeEach(() => useMetricMetadata.mockReset());

  it("shows the description, type, unit and sources from the catalog", () => {
    useMetricMetadata.mockReturnValue({
      isLoading: false,
      data: {
        name: "http_request_duration_seconds",
        type: "histogram",
        help: "Duration of HTTP requests.",
        unit: "seconds",
        sources: ["source-a"],
      },
    });
    render(<MetricDetailHeader metricName="http_request_duration_seconds" />);

    expect(screen.getByText("Duration of HTTP requests.")).toBeInTheDocument();
    expect(screen.getByText("seconds")).toBeInTheDocument();
    expect(screen.getByText("source-a")).toBeInTheDocument();
    expect(useMetricMetadata).toHaveBeenCalledWith(
      "http_request_duration_seconds",
    );
  });

  it("says so when the catalog doesn't know the metric", () => {
    useMetricMetadata.mockReturnValue({ isLoading: false, data: null });
    render(<MetricDetailHeader metricName="unknown_metric" />);
    expect(
      screen.getByText("Not in the metrics catalog yet"),
    ).toBeInTheDocument();
  });
});
