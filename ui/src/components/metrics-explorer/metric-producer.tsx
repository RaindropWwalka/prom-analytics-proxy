"use client";

import * as React from "react";
import { type SortingState } from "@tanstack/react-table";
import { type LegacyColumnDef } from "@tanstack/react-table/legacy";
import { ArrowUpDown } from "lucide-react";
import { Progress } from "@/components/ui/progress";
import { Producer } from "@/lib/types";
import { DataTable } from "@/components/data-table";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  CardDescription,
} from "@/components/ui/card";

import { useMetricStatistics } from "@/app/metrics/use-metrics-data";
import { useDateRange } from "@/contexts/date-range";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const ALL_SOURCES = "__all__";

interface MetricProducersProps {
  metricName: string;
}

export function MetricProducers({ metricName }: MetricProducersProps) {
  const { dateRange } = useDateRange();
  const { data, isLoading } = useMetricStatistics(metricName, dateRange);
  const allProducers: Producer[] = React.useMemo(
    () => data.statistics?.producers || [],
    [data.statistics],
  );
  // Local sorting state must be declared unconditionally to respect Hooks rules
  const [sorting, setSorting] = React.useState<SortingState>([]);
  const [sourceFilter, setSourceFilter] = React.useState(ALL_SOURCES);

  // Producers are split by source only when the proxy knows which
  // Prometheus each series came from (producers_source_label).
  const sources = React.useMemo(
    () =>
      [
        ...new Set(allProducers.map((p) => p.source).filter(Boolean)),
      ].sort() as string[],
    [allProducers],
  );
  const producers = React.useMemo(
    () =>
      sourceFilter === ALL_SOURCES
        ? allProducers
        : allProducers.filter((p) => p.source === sourceFilter),
    [allProducers, sourceFilter],
  );

  // Calculate total series for percentage calculations (empty while loading)
  const totalSeries = producers.reduce(
    (sum, producer) => sum + producer.series,
    0,
  );

  const columns = React.useMemo<LegacyColumnDef<Producer>[]>(
    () => [
      {
        accessorKey: "job",
        header: ({ column }) => {
          return (
            <div
              className="flex cursor-pointer items-center"
              onClick={() =>
                column.toggleSorting(column.getIsSorted() === "asc")
              }
            >
              Producer
              <ArrowUpDown className="ml-2 h-4 w-4" />
            </div>
          );
        },
        cell: ({ row }) => (
          <div className="font-medium">{row.getValue("job")}</div>
        ),
      },
      ...(sources.length > 0
        ? [
            {
              accessorKey: "source",
              header: "Source",
              cell: ({ row }) => (
                <div className="truncate text-muted-foreground">
                  {row.original.source || "—"}
                </div>
              ),
            } satisfies LegacyColumnDef<Producer>,
          ]
        : []),
      {
        id: "contribution",
        header: () => <div className="text-right">Contribution</div>,
        cell: ({ row }) => {
          const series = row.getValue("series") as number;
          const percentage = Math.round((series / totalSeries) * 100);
          return (
            <div className="flex items-center gap-2 justify-end">
              <Progress value={percentage} className="h-1.5" />
              <span className="text-xs text-muted-foreground">
                {percentage}%
              </span>
            </div>
          );
        },
      },
      {
        accessorKey: "series",
        header: ({ column }) => {
          return (
            <div
              className="flex cursor-pointer items-center justify-end"
              onClick={() =>
                column.toggleSorting(column.getIsSorted() === "asc")
              }
            >
              Series
              <ArrowUpDown className="ml-2 h-4 w-4" />
            </div>
          );
        },
        cell: ({ row }) => {
          const series = row.getValue("series") as number;
          return <div className="text-right">{series.toLocaleString()}</div>;
        },
      },
    ],
    [totalSeries, sources.length],
  );

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>
            <Skeleton className="h-4 w-32" />
          </CardTitle>
          <CardDescription>
            <Skeleton className="h-3 w-full" />
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Skeleton className="h-[300px] w-full" />
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Producers</CardTitle>
        <CardDescription>
          Metric producers (also called Jobs) are services or applications that
          generate and expose Prometheus metrics.
          <br />
          Each producer contributes to the total number of time series in your
          monitoring system. The same job scraped by several Prometheus is shown
          once per Prometheus.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        {sources.length > 0 && (
          <Select value={sourceFilter} onValueChange={setSourceFilter}>
            <SelectTrigger className="sm:w-[320px]" aria-label="Source">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL_SOURCES}>All sources</SelectItem>
              {sources.map((source) => (
                <SelectItem key={source} value={source}>
                  {source}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
        <DataTable
          data={producers}
          columns={columns}
          searchColumn="job"
          pagination={true}
          pageSize={10}
          sortingState={sorting}
          onSortingChange={setSorting}
        />
      </CardContent>
    </Card>
  );
}
