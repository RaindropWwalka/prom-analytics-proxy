import { render, screen } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import { QueryOrigins } from "./query-origins";

describe("QueryOrigins", () => {
  it("counts runs per kind, the rest being other clients", () => {
    render(
      <QueryOrigins
        origins={{ executions: 10, dashboardRuns: 6, alertRuns: 3 }}
      />,
    );
    expect(screen.getByText("Dashboards 6")).toBeInTheDocument();
    expect(screen.getByText("Alerts 3")).toBeInTheDocument();
    expect(screen.getByText("Other 1")).toBeInTheDocument();
  });

  it("lists who ran it when open", () => {
    render(
      <QueryOrigins
        origins={{
          executions: 3,
          dashboardRuns: 2,
          alertRuns: 1,
          dashboards: ["Service overview"],
          rules: ["TargetDown"],
        }}
        open={true}
        onOpenChange={() => {}}
      />,
    );
    expect(screen.getByText("Service overview")).toBeInTheDocument();
    expect(screen.getByText("TargetDown")).toBeInTheDocument();
    expect(screen.queryByText("Other clients")).not.toBeInTheDocument();
  });

  it("shows a dash when nothing ran it", () => {
    render(<QueryOrigins origins={{ executions: 0 }} />);
    expect(screen.getByText("—")).toBeInTheDocument();
  });
});
