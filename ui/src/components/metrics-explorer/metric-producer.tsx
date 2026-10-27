"use client";

import * as React from "react";
import { type SortingState } from "@tanstack/react-table";
import { type LegacyColumnDef } from "@tanstack/react-table/legacy";
import { Progress } from "@/components/ui/progress";
import { Producer } from "@/lib/types";
import { DataTable, DataTableColumnHeader } from "@/components/data-table";
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  CardDescription,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useMetricStatistics } from "@/app/metrics/use-metrics-data";
import { useDateRange } from "@/contexts/date-range";
import { Skeleton } from "@/components/ui/skeleton";
import {
  filterProducers,
  NO_PRODUCER_FILTERS,
  type ProducerFilters,
} from "./producer-filters";

// Producer columns that can outgrow their cell: the table shows their whole
// value on hover.
type TextColumn = "job" | "service" | "source";

const TEXT_COLUMNS: { key: TextColumn; title: string }[] = [
  { key: "job", title: "Job" },
  { key: "service", title: "Service" },
  { key: "source", title: "Source" },
];

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
  const [sorting, setSorting] = React.useState<SortingState>([
    { id: "series", desc: true },
  ]);
  const [filters, setFilters] =
    React.useState<ProducerFilters>(NO_PRODUCER_FILTERS);

  // Only the columns the producers carry: source with
  // producers_source_label, service with producers_service_label.
  const shown = React.useMemo(
    () =>
      TEXT_COLUMNS.filter(
        ({ key }) => key === "job" || allProducers.some((p) => !!p[key]),
      ),
    [allProducers],
  );
  const producers = React.useMemo(
    () => filterProducers(allProducers, filters),
    [allProducers, filters],
  );

  // Contribution is a share of everything the metric has, not of what the
  // filters leave, so it doesn't change as filters narrow the table.
  const totalSeries = allProducers.reduce((sum, p) => sum + p.series, 0);

  const columns = React.useMemo<
    Array<LegacyColumnDef<Producer> & { maxWidth?: number }>
  >(
    () => [
      ...shown.map(
        ({
          key,
          title,
        }): LegacyColumnDef<Producer> & { maxWidth?: number } => ({
          accessorKey: key,
          maxWidth: 320,
          header: ({ column }) => (
            <DataTableColumnHeader column={column} title={title} />
          ),
          cell: ({ row }) => (
            <span className={key === "job" ? "font-medium" : undefined}>
              {row.original[key] || "—"}
            </span>
          ),
        }),
      ),
      {
        id: "contribution",
        size: 180,
        header: () => <div className="text-right">Contribution</div>,
        cell: ({ row }) => {
          const percentage = totalSeries
            ? Math.round((row.original.series / totalSeries) * 100)
            : 0;
          return (
            <div className="flex items-center gap-2 justify-end">
              <Progress value={percentage} className="h-1.5" />
              <span className="w-10 text-right text-xs text-muted-foreground">
                {percentage}%
              </span>
            </div>
          );
        },
      },
      {
        accessorKey: "series",
        size: 110,
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title="Series" />
        ),
        cell: ({ row }) => (
          <div className="text-right tabular-nums">
            {row.original.series.toLocaleString()}
          </div>
        ),
      },
    ],
    [shown, totalSeries],
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

  const filtered = producers.length !== allProducers.length;

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
          once per Prometheus, and a job carrying many services once per
          service.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
          {shown.map(({ key, title }) => (
            <Input
              key={key}
              aria-label={`Filter ${title}`}
              placeholder={`${title} contains…`}
              value={filters[key]}
              onChange={(e) =>
                setFilters((prev) => ({ ...prev, [key]: e.target.value }))
              }
            />
          ))}
          <Input
            type="number"
            min={0}
            aria-label="Series at least"
            placeholder="Series ≥"
            value={filters.minSeries}
            onChange={(e) =>
              setFilters((prev) => ({ ...prev, minSeries: e.target.value }))
            }
          />
        </div>
        <div className="flex items-center justify-between text-sm text-muted-foreground">
          <span>
            {filtered
              ? `${producers.length} of ${allProducers.length} producers`
              : `${allProducers.length} producers`}
          </span>
          {filtered && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setFilters(NO_PRODUCER_FILTERS)}
              className="cursor-pointer"
            >
              Reset filters
            </Button>
          )}
        </div>
        <DataTable
          data={producers}
          columns={columns}
          pagination={true}
          pageSize={10}
          sortingState={sorting}
          onSortingChange={setSorting}
        />
      </CardContent>
    </Card>
  );
}
