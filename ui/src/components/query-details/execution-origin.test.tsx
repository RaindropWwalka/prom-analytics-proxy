import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect } from "vitest";
import { ExecutionOrigin, HeadersPopover } from "./execution-origin";
import { executionOrigin } from "./execution-origin-utils";

describe("executionOrigin", () => {
  it("prefers alerting over the dashboard an alert was built from", () => {
    expect(
      executionOrigin({
        fromalert: "true",
        "x-rule-name": "HighErrorRate",
        "x-rule-folder": "team-a",
        "x-dashboard-uid": "dash-a",
      }),
    ).toEqual({ kind: "alert", rule: "HighErrorRate", folder: "team-a" });
  });

  it("names a dashboard by title, falling back to its uid", () => {
    expect(
      executionOrigin({ "x-dashboard-uid": "dash-a", "x-panel-id": "3" }),
    ).toEqual({ kind: "dashboard", dashboard: "dash-a", panel: "panel 3" });
    expect(
      executionOrigin({
        "x-dashboard-uid": "dash-a",
        "x-dashboard-title": "Service overview",
        "x-panel-title": "Error rate",
      }),
    ).toEqual({
      kind: "dashboard",
      dashboard: "Service overview",
      panel: "Error rate",
    });
  });

  it("falls back to the client name, then the user agent", () => {
    expect(
      executionOrigin({
        "user-agent": "curl/8.7.1",
        "x-client-name": "team-a/report",
      }),
    ).toEqual({ kind: "client", client: "team-a/report" });
    expect(executionOrigin({ "user-agent": "curl/8.7.1" })).toEqual({
      kind: "client",
      client: "curl/8.7.1",
    });
  });

  it("is unknown without headers", () => {
    expect(executionOrigin(undefined)).toEqual({ kind: "unknown" });
    expect(executionOrigin({})).toEqual({ kind: "unknown" });
  });
});

describe("ExecutionOrigin", () => {
  it("shows the rule of an alert evaluation", () => {
    render(
      <ExecutionOrigin
        headers={{ fromalert: "true", "x-rule-name": "HighErrorRate" }}
      />,
    );
    expect(screen.getByText("Alert")).toBeInTheDocument();
    expect(screen.getByText("HighErrorRate")).toBeInTheDocument();
  });
});

describe("HeadersPopover", () => {
  it("lists every stored header on click", async () => {
    render(
      <HeadersPopover
        headers={{
          "user-agent": "Grafana/12.0.0",
          "x-panel-id": "3",
          empty: "",
        }}
      />,
    );
    await userEvent.click(screen.getByRole("button", { name: "2 headers" }));
    expect(screen.getByText("user-agent")).toBeInTheDocument();
    expect(screen.getByText("Grafana/12.0.0")).toBeInTheDocument();
    expect(screen.getByText("x-panel-id")).toBeInTheDocument();
    expect(screen.queryByText("empty")).not.toBeInTheDocument();
  });

  it("shows a dash without headers", () => {
    render(<HeadersPopover headers={{}} />);
    expect(screen.getByText("—")).toBeInTheDocument();
  });
});
