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
import { formatUnit } from "@/lib/utils";
import {
  catalogLink,
  groupLabel,
  groupTitle,
  totals,
  unusedPercent,
} from "./analysis-utils";

// How many groups the chart shows; the table has them all.
const CHART_GROUPS = 15;

type Row = UnusedAnalysisRow & {
  label: string;
  used: number;
  usedSeries: number;
  pct: number;
};

type Measure = "metrics" | "series";

export default function AnalysisPage() {
  const [, setLocation] = useLocation();
  const [groupBy, setGroupBy] = useState<UnusedAnalysisGroupBy>("job");
  // What the chart ranks groups by: unused metrics, or the series they hold.
  const [measure, setMeasure] = useState<Measure>("series");
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
        usedSeries: r.series - r.unusedSeries,
        pct: unusedPercent(r),
      })),
    [data, shownGroupBy],
  );
  const chartRows = useMemo(
    () =>
      [...rows].sort((a, b) =>
        measure === "series"
          ? b.unusedSeries - a.unusedSeries
          : b.unused - a.unused,
      ),
    [rows, measure],
  );
  const hasSeries = rows.some((r) => r.series > 0);
  const sum = totals(rows);
  const seriesSum = rows.reduce(
    (acc, r) => ({
      series: acc.series + r.series,
      unused: acc.unused + r.unusedSeries,
    }),
    { series: 0, unused: 0 },
  );

  const columns: LegacyColumnDef<Row>[] = [
    {
      accessorKey: "label",
      size: 320,
      header: ({ column }) => (
        <DataTableColumnHeader
          column={column}
          title={groupTitle(shownGroupBy)}
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
      accessorKey: "series",
      size: 110,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Series" />
      ),
      cell: ({ row }) => (
        <span className="tabular-nums">{formatUnit(row.original.series)}</span>
      ),
    },
    {
      accessorKey: "unusedSeries",
      size: 130,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Unused series" />
      ),
      cell: ({ row }) => (
        <span className="tabular-nums">
          {formatUnit(row.original.unusedSeries)}
        </span>
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
            <SelectItem value="service">By service</SelectItem>
            <SelectItem value="source">By source</SelectItem>
            <SelectItem value="job,source">By job and source</SelectItem>
            <SelectItem value="service,source">
              By service and source
            </SelectItem>
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
          Nothing to analyze. Grouping by job or service needs the job index
          (inventory.job_sync_enabled, with inventory.service_label for
          services), and by source needs inventory.metadata_sources.
        </p>
      )}

      {rows.length > 0 && (
        <>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <SummaryCard title="Metrics" value={sum.total} />
            <SummaryCard title="Unused metrics" value={sum.unused} />
            <SummaryCard
              title="Unused share"
              value={`${sum.total > 0 ? Math.round((sum.unused / sum.total) * 100) : 0}%`}
            />
            <SummaryCard
              title="Unused series"
              value={
                hasSeries
                  ? `${formatUnit(seriesSum.unused)} / ${formatUnit(seriesSum.series)}`
                  : "—"
              }
            />
          </div>

          <Card>
            <CardHeader>
              <div className="flex flex-wrap items-center justify-between gap-2">
                <CardTitle>
                  {measure === "series"
                    ? "Most series in unused metrics"
                    : "Most unused metrics"}
                </CardTitle>
                <Select
                  value={measure}
                  onValueChange={(v) => setMeasure(v as Measure)}
                >
                  <SelectTrigger className="w-[160px]" aria-label="Measure">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="series">By series</SelectItem>
                    <SelectItem value="metrics">By metrics</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <CardDescription>
                Top {Math.min(CHART_GROUPS, rows.length)} groups by unused{" "}
                {measure}.
                {measure === "series" &&
                  !hasSeries &&
                  " No series recorded yet: needs inventory.cardinality_limit and metadata_sources."}
                {measure === "series" &&
                  hasSeries &&
                  " Series are each source's head series; a metric shared by several groups counts under each."}
                {shownGroupBy.endsWith(",source") &&
                  " A metric counts under every source reporting it."}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer
                width="100%"
                height={Math.max(200, Math.min(CHART_GROUPS, rows.length) * 32)}
              >
                <BarChart
                  data={chartRows.slice(0, CHART_GROUPS)}
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
                    dataKey={measure === "series" ? "unusedSeries" : "unused"}
                    name="Unused"
                    stackId="metrics"
                    fill="var(--chart-1)"
                  />
                  <Bar
                    dataKey={measure === "series" ? "usedSeries" : "used"}
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
