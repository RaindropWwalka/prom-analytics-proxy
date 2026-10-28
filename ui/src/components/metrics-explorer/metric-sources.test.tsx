import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { MetricSources } from "./metric-sources";

describe("MetricSources", () => {
  it("shows a dash when no source reports the metric", () => {
    render(<MetricSources sources={[]} />);
    expect(screen.getByText("—")).toBeInTheDocument();
  });

  it("shows a dash when sources are absent", () => {
    render(<MetricSources />);
    expect(screen.getByText("—")).toBeInTheDocument();
  });

  it("shows every source when they fit", () => {
    render(<MetricSources sources={["source-a", "source-b"]} />);
    expect(screen.getByText("source-a")).toBeInTheDocument();
    expect(screen.getByText("source-b")).toBeInTheDocument();
    expect(screen.queryByText(/^\+/)).not.toBeInTheDocument();
  });

  it("collapses the rest into a count, keeping all of them in the tooltip", () => {
    const { container } = render(
      <MetricSources sources={["a", "b", "c", "d"]} max={2} />,
    );
    expect(screen.getByText("a")).toBeInTheDocument();
    expect(screen.getByText("b")).toBeInTheDocument();
    expect(screen.queryByText("c")).not.toBeInTheDocument();
    expect(screen.getByText("+2")).toBeInTheDocument();
    // Checked on the attribute itself: title queries collapse the newlines.
    expect(container.firstChild).toHaveAttribute("title", "a\nb\nc\nd");
  });
});
