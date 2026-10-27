import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { MetricDescription } from "./metric-description";

const metric = {
  name: "http_requests_total",
  type: "counter",
  help: "Total number of HTTP requests handled, partitioned by status code and method.",
  unit: "requests",
  sources: ["source-a", "source-b"],
};

describe("MetricDescription", () => {
  it("shows a dash when the metric has no help", () => {
    render(<MetricDescription metric={{ ...metric, help: "" }} />);
    expect(screen.getByText("—")).toBeInTheDocument();
  });

  it("opens the full help with type, unit and sources on click", async () => {
    render(<MetricDescription metric={metric} />);

    await userEvent.click(screen.getByRole("button"));

    expect(screen.getByText("requests")).toBeInTheDocument();
    expect(screen.getByText("source-a")).toBeInTheDocument();
    expect(screen.getByText("source-b")).toBeInTheDocument();
    expect(screen.getAllByText(metric.help)).toHaveLength(2);
  });

  it("doesn't let the click reach the row, which opens the metric page", async () => {
    const onRowClick = vi.fn();
    render(
      <div onClick={onRowClick}>
        <MetricDescription metric={metric} />
      </div>,
    );

    await userEvent.click(screen.getByRole("button"));

    expect(onRowClick).not.toHaveBeenCalled();
  });
});
