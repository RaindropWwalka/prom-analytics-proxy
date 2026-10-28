"use client";

import * as React from "react";
import { SortingState } from "@tanstack/react-table";
import { type LegacyColumnDef } from "@tanstack/react-table/legacy";
import { MetricTypeTag } from "@/components/metrics-explorer/metric-type-tag";
import { MetricSources } from "@/components/metrics-explorer/metric-sources";
import { MetricDescription } from "@/components/metrics-explorer/metric-description";
import { useLocation } from "wouter";
import { MetricMetadata, PagedResult, TableState } from "@/lib/types";
import { fromUTC } from "@/lib/utils/date-utils";
import { DataTable, DataTableColumnHeader } from "@/components/data-table";
import { ROUTES } from "@/lib/routes";
import { formatUnit } from "@/lib/utils";
import { seriesBreakdown, totalSeries } from "@/lib/utils/series";

interface MetricsTableProps {
  metrics?: PagedResult<MetricMetadata>;
  searchQuery: string;
  tableState: TableState;
  onTableStateChange: (state: TableState) => void;
}

export function MetricsTable({
  metrics,
  searchQuery,
  tableState,
  onTableStateChange,
}: MetricsTableProps) {
  const [, setLocation] = useLocation();
  // The one popover open in the table, if any: "<kind>:<metric>".
  const [openPopover, setOpenPopover] = React.useState<string | null>(null);
  const popover = (key: string) => ({
    open: openPopover === key,
    onOpenChange: (open: boolean) => setOpenPopover(open ? key : null),
  });
  const prevSearchRef = React.useRef(searchQuery);

  // Update filter when search query changes
  React.useEffect(() => {
    if (prevSearchRef.current !== searchQuery) {
      prevSearchRef.current = searchQuery;
      onTableStateChange({
        ...tableState,
        filter: searchQuery,
        page: 1, // Reset to first page only when search changes
      });
    }
  }, [searchQuery, tableState, onTableStateChange]);

  const handleMetricClick = (metricName: string) => {
    // Use the route path from ROUTES constant, replacing the parameter with the actual value
    const detailsPath = ROUTES.METRICS_DETAILS.replace(
      ":metric",
      encodeURIComponent(metricName),
    );
    setLocation(detailsPath);
  };

  // Initialize sorting state from tableState
  const sortingState: SortingState = React.useMemo(() => {
    return tableState.sortBy
      ? [
          {
            id: tableState.sortBy,
            desc: tableState.sortOrder === "desc",
          },
        ]
      : [];
  }, [tableState.sortBy, tableState.sortOrder]);

  // Handler functions for DataTable callbacks
  const handleSortingChange = (newSorting: SortingState) => {
    if (newSorting.length > 0) {
      onTableStateChange({
        ...tableState,
        sortBy: newSorting[0].id,
        sortOrder: newSorting[0].desc ? "desc" : "asc",
      });
    }
  };

  const handleFilterChange = (value: string) => {
    onTableStateChange({
      ...tableState,
      filter: value,
      page: 1, // Reset to first page on filter change
    });
  };

  const handlePaginationChange = (page: number) => {
    onTableStateChange({
      ...tableState,
      page,
    });
  };

  // Name and description have no fixed width: they share whatever the
  // narrow columns leave, so the table follows the window's width.
  const columns: LegacyColumnDef<MetricMetadata>[] = [
    {
      accessorKey: "name",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Metric Name" />
      ),
      cell: ({ row }) => {
        return (
          <div
            className="font-medium truncate cursor-pointer hover:text-blue-500"
            title={row.getValue("name")}
            onClick={(e) => {
              e.stopPropagation();
              handleMetricClick(row.getValue("name"));
            }}
          >
            {row.getValue("name")}
          </div>
        );
      },
    },
    {
      accessorKey: "type",
      size: 100,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Type" />
      ),
      cell: ({ row }) => <MetricTypeTag type={row.getValue("type")} />,
    },
    {
      accessorKey: "help",
      header: "Description",
      cell: ({ row }) => (
        <MetricDescription
          metric={row.original}
          {...popover(`description:${row.original.name}`)}
        />
      ),
    },
    {
      accessorKey: "sources",
      size: 150,
      header: "Sources",
      enableSorting: false,
      cell: ({ row }) => (
        <MetricSources
          sources={row.original.sources}
          {...popover(`sources:${row.original.name}`)}
        />
      ),
    },
    {
      id: "series",
      size: 90,
      enableSorting: false,
      header: "Series",
      cell: ({ row }) => {
        const series = row.original.series;
        const total = totalSeries(series);
        return total === undefined ? (
          <span className="text-muted-foreground">—</span>
        ) : (
          <span className="tabular-nums" title={seriesBreakdown(series)}>
            {formatUnit(total)}
          </span>
        );
      },
    },
    {
      accessorKey: "alertCount",
      size: 70,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Alerts" />
      ),
      cell: ({ row }) => (
        <span className="tabular-nums">{row.getValue("alertCount") ?? 0}</span>
      ),
    },
    {
      accessorKey: "recordCount",
      size: 80,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Records" />
      ),
      cell: ({ row }) => (
        <span className="tabular-nums">{row.getValue("recordCount") ?? 0}</span>
      ),
    },
    {
      accessorKey: "dashboardCount",
      size: 100,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Dashboards" />
      ),
      cell: ({ row }) => (
        <span className="tabular-nums">
          {row.getValue("dashboardCount") ?? 0}
        </span>
      ),
    },
    {
      accessorKey: "queryCount",
      size: 110,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title="Queries (30d)" />
      ),
      cell: ({ row }) => {
        const count = row.getValue<number>("queryCount") ?? 0;
        const last = row.original.lastQueriedAt;
        return (
          <div className="flex items-center gap-2">
            <span className="tabular-nums">{count}</span>
            {last && (
              <span
                className="text-xs text-muted-foreground"
                title={`Last queried at ${fromUTC(last).toLocaleString()}`}
              >
                •
              </span>
            )}
          </div>
        );
      },
    },
  ];

  return (
    <div className="mt-4">
      <DataTable<MetricMetadata>
        data={metrics?.data || []}
        columns={columns}
        pagination={true}
        pageSize={tableState.pageSize}
        className="w-full"
        serverSide={true}
        sortingState={sortingState}
        filterValue={tableState.filter}
        currentPage={tableState.page}
        totalPages={metrics?.totalPages || 1}
        onSortingChange={handleSortingChange}
        onFilterChange={handleFilterChange}
        onPaginationChange={handlePaginationChange}
        onPageSizeChange={(pageSize) =>
          onTableStateChange({ ...tableState, pageSize, page: 1 })
        }
        onRowClick={(row) => handleMetricClick(row.name)}
      />
    </div>
  );
}
