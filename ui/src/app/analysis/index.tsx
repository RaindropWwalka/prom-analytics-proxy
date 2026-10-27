import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useLocation } from "wouter";
import { type LegacyColumnDef } from "@tanstack/react-table/legacy";
import {
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { Play } from "lucide-react";
import {
  analyzeUnused,
  type UnusedAnalysisGroupBy,
  type UnusedAnalysisRow,
} from "@/api/metrics";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { DataTable, DataTableColumnHeader } from "@/components/data-table";
import {
  catalogLink,
  groupLabel,
  totals,
  unusedPercent,
} from "./analysis-utils";

// How many groups the chart shows; the table has them all.
const CHART_GROUPS = 15;

type Row = UnusedAnalysisRow & { label: string; used: number; pct: number };

export default function AnalysisPage() {
  const [, setLocation] = useLocation();
  const [groupBy, setGroupBy] = useState<UnusedAnalysisGroupBy>("job");
  // The analysis aggregates the whole catalog, so it runs when asked to,
  // not on every visit.
  const [requested, setRequested] = useState<UnusedAnalysisGroupBy | null>(
    null,
  );

  const { data, isFetching, error, refetch } = useQuery({
    queryKey: ["unusedAnalysis", requested],
    queryFn: () => analyzeUnused(requested as UnusedAnalysisGroupBy),
    enabled: requested !== null,
  });

  const run = () => {
    if (requested === groupBy) {
      void refetch();
    } else {
      setRequested(groupBy);
    }
  };

  const shownGroupBy = requested ?? groupBy;
  const rows: Row[] = useMemo(
    () =>
      (data ?? []).map((r) => ({
        ...r,
        label: groupLabel(r, shownGroupBy),
        used: r.total - r.unused,
        pct: unusedPercent(r),
      })),
    [data, shownGroupBy],
  );
  const sum = totals(rows);

  const columns: LegacyColumnDef<Row>[] = [
    {
      accessorKey: "label",
      size: 320,
      header: ({ column }) => (
        <DataTableColumnHeader
          column={column}
          title={
            shownGroupBy === "job"
              ? "Job"
              : shownGroupBy === "source"
                ? "Source"
                : "Job @ Source"
          }
        />
      ),
      cell: ({ row }) => (
        <span className="block truncate font-medium" title={row.original.label}>
          {row.original.label}
        </span>
      ),
    },
    {
      accessorKey: "total",
      size: 110,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Metrics" />
      ),
      cell: ({ row }) => (
        <span className="tabular-nums">{row.original.total}</span>
      ),
    },
    {
      accessorKey: "unused",
      size: 110,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Unused" />
      ),
      cell: ({ row }) => (
        <span className="tabular-nums">{row.original.unused}</span>
      ),
    },
    {
      accessorKey: "pct",
      size: 200,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Unused %" />
      ),
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <Progress value={row.original.pct} className="h-1.5" />
          <span className="w-10 text-right text-xs tabular-nums text-muted-foreground">
            {row.original.pct}%
          </span>
        </div>
      ),
    },
  ];

  return (
    <div className="space-y-4 p-4">
      <div>
        <h1 className="text-2xl font-bold">Unused Metrics Analysis</h1>
        <p className="text-sm text-muted-foreground">
          How many of each group&apos;s metrics no query, dashboard, alert or
          recording rule uses. Click a row to see them in the catalog.
        </p>
      </div>

      <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
        <Select
          value={groupBy}
          onValueChange={(v) => setGroupBy(v as UnusedAnalysisGroupBy)}
        >
          <SelectTrigger className="sm:w-[200px]" aria-label="Group by">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="job">By job</SelectItem>
            <SelectItem value="source">By source</SelectItem>
            <SelectItem value="job,source">By job and source</SelectItem>
          </SelectContent>
        </Select>
        <Button onClick={run} disabled={isFetching} className="cursor-pointer">
          <Play className="h-4 w-4" />
          {isFetching ? "Analyzing…" : "Run analysis"}
        </Button>
      </div>

      {error && (
        <p className="text-sm text-destructive">{(error as Error).message}</p>
      )}

      {data && rows.length === 0 && (
        <p className="text-sm text-muted-foreground">
          Nothing to analyze. Grouping by job needs the job index
          (inventory.job_sync_enabled), and by source needs
          inventory.metadata_sources.
        </p>
      )}

      {rows.length > 0 && (
        <>
          <div className="grid gap-4 sm:grid-cols-3">
            <SummaryCard title="Metrics" value={sum.total} />
            <SummaryCard title="Unused" value={sum.unused} />
            <SummaryCard
              title="Unused share"
              value={`${sum.total > 0 ? Math.round((sum.unused / sum.total) * 100) : 0}%`}
            />
          </div>

          <Card>
            <CardHeader>
              <CardTitle>Most unused metrics</CardTitle>
              <CardDescription>
                Top {Math.min(CHART_GROUPS, rows.length)} groups by unused
                metrics.
                {shownGroupBy === "job,source" &&
                  " A metric of a job counts under every source reporting it."}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer
                width="100%"
                height={Math.max(200, Math.min(CHART_GROUPS, rows.length) * 32)}
              >
                <BarChart
                  data={rows.slice(0, CHART_GROUPS)}
                  layout="vertical"
                  margin={{ left: 8, right: 16 }}
                >
                  <CartesianGrid strokeDasharray="3 3" horizontal={false} />
                  <XAxis type="number" fontSize={12} />
                  <YAxis
                    type="category"
                    dataKey="label"
                    width={220}
                    fontSize={12}
                    tickFormatter={(v: string) =>
                      v.length > 32 ? `${v.slice(0, 31)}…` : v
                    }
                  />
                  <Tooltip />
                  <Legend />
                  <Bar
                    dataKey="unused"
                    name="Unused"
                    stackId="metrics"
                    fill="var(--chart-1)"
                  />
                  <Bar
                    dataKey="used"
                    name="Used"
                    stackId="metrics"
                    fill="var(--chart-3)"
                  />
                </BarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>

          <DataTable<Row>
            data={rows}
            columns={columns}
            pagination={true}
            pageSize={20}
            searchColumn="label"
            onRowClick={(row) => setLocation(catalogLink(row))}
          />
        </>
      )}
    </div>
  );
}

function SummaryCard({
  title,
  value,
}: {
  title: string;
  value: number | string;
}) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardDescription>{title}</CardDescription>
        <CardTitle className="text-2xl tabular-nums">{value}</CardTitle>
      </CardHeader>
    </Card>
  );
}
