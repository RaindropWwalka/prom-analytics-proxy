export type Origin =
  | { kind: "alert"; rule?: string; folder?: string }
  | { kind: "dashboard"; dashboard: string; panel?: string }
  | { kind: "client"; client: string }
  | { kind: "unknown" };

// executionOrigin tells who sent a query from the headers stored with it.
// Grafana alerting marks every evaluation with FromAlert, and an alert built
// from a dashboard panel carries the dashboard's uid too, so alerting wins.
export function executionOrigin(headers?: Record<string, string>): Origin {
  const h = headers ?? {};
  if (h["fromalert"] === "true") {
    return {
      kind: "alert",
      rule: h["x-rule-name"],
      folder: h["x-rule-folder"],
    };
  }
  const dashboard = h["x-dashboard-title"] || h["x-dashboard-uid"];
  if (dashboard) {
    const panel =
      h["x-panel-title"] || (h["x-panel-id"] && `panel ${h["x-panel-id"]}`);
    return { kind: "dashboard", dashboard, panel: panel || undefined };
  }
  const client = h["x-client-name"] || h["user-agent"];
  if (client) {
    return { kind: "client", client };
  }
  return { kind: "unknown" };
}
