import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
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

describe("MetricDescription in a table", () => {
  // The table's part: it remembers which one description is open.
  function TwoDescriptions() {
    const [open, setOpen] = useState<string | null>(null);
    return (
      <>
        {["first_metric", "second_metric"].map((name) => (
          <MetricDescription
            key={name}
            metric={{ ...metric, name, help: `help of ${name}` }}
            open={open === name}
            onOpenChange={(o) => setOpen(o ? name : null)}
          />
        ))}
      </>
    );
  }

  it("keeps only one description open at a time", async () => {
    render(<TwoDescriptions />);

    await userEvent.click(
      screen.getByRole("button", { name: "help of first_metric" }),
    );
    expect(screen.getByText("first_metric")).toBeInTheDocument();

    await userEvent.click(
      screen.getByRole("button", { name: "help of second_metric" }),
    );
    expect(screen.getByText("second_metric")).toBeInTheDocument();
    expect(screen.queryByText("first_metric")).not.toBeInTheDocument();
  });
});
