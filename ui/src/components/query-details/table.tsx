import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { DataTable, DataTableColumnHeader } from "@/components/data-table";
import type { SortingState } from "@tanstack/react-table";
import type { LegacyColumnDef } from "@tanstack/react-table/legacy";
import type { TableData } from "@/components/data-table/types";
import { useDateRange } from "@/contexts/date-range";
import { getQueryExecutions } from "@/api/queries";
import type { PagedResult, QueryExecution } from "@/lib/types";
import { formatUTCtoLocal } from "@/lib/utils/date-utils";
import { Badge } from "@/components/ui/badge";
import { formatUnit } from "@/lib/utils";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useSearchParams } from "wouter";
import { ExecutionOrigin, HeadersPopover } from "./execution-origin";
import { ExecutionsFilters } from "./executions-filters";
import { useExecutionsFilters } from "./use-executions-filters";

type ExtendedColumnDef<
  TData extends TableData,
  TValue = unknown,
> = LegacyColumnDef<TData, TValue> & {
  maxWidth?: string | number;
};

// Format time range duration in human-readable form
const formatTimeRangeDuration = (start: string, end: string): string => {
  try {
    const startTime = new Date(start).getTime();
    const endTime = new Date(end).getTime();
    const diffMs = endTime - startTime;

    if (diffMs < 0) return "-";

    const seconds = Math.floor(diffMs / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    const days = Math.floor(hours / 24);

    if (days > 0) {
      const remainingHours = hours % 24;
      return remainingHours > 0 ? `${days}d ${remainingHours}h` : `${days}d`;
    }
    if (hours > 0) {
      const remainingMinutes = minutes % 60;
      return remainingMinutes > 0
        ? `${hours}h ${remainingMinutes}m`
        : `${hours}h`;
    }
    if (minutes > 0) {
      const remainingSeconds = seconds % 60;
      return remainingSeconds > 0
        ? `${minutes}m ${remainingSeconds}s`
        : `${minutes}m`;
    }
    if (seconds > 0) {
      return `${seconds}s`;
    }
    return `${diffMs}ms`;
  } catch {
    return "-";
  }
};

// Which row's headers are open; the table keeps one open at a time.
interface HeadersPopoverState {
  isOpen: (row: number) => boolean;
  setOpen: (row: number, open: boolean) => void;
}

const buildColumns = (
  headers: HeadersPopoverState,
): ExtendedColumnDef<QueryExecution>[] => [
  {
    accessorKey: "timestamp",
    size: 170,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Timestamp" />
    ),
    cell: ({ row }) => {
      const value = String(row.getValue("timestamp"));
      try {
        return (
          <div className="flex items-baseline gap-2">
            <span className="font-medium">
              {formatUTCtoLocal(value, "dd/MM/yyyy")}
            </span>
            <span className="text-xs text-muted-foreground">
              {formatUTCtoLocal(value, "HH:mm")}
            </span>
          </div>
        );
      } catch {
        return "-";
      }
    },
  },
  {
    accessorKey: "type",
    size: 110,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Type" />
    ),
    cell: ({ row }) => {
      const t = String(row.getValue("type"));
      const label = t === "range" ? "Range" : "Instant";
      return <Badge variant="outline">{label}</Badge>;
    },
  },
  {
    accessorKey: "status",
    size: 105,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Status" />
    ),
    cell: ({ row }) => {
      const code = Number(row.getValue("status"));
      const isSuccess = code >= 200 && code < 300;
      const isTimeout = code === 0 || code === 408 || code === 504;
      const classes = isSuccess
        ? "bg-emerald-100 text-emerald-700 border-emerald-200"
        : isTimeout
          ? "bg-amber-100 text-amber-700 border-amber-200"
          : "bg-red-100 text-red-700 border-red-200";
      return (
        <Badge variant="outline" className={classes}>
          <span className="font-mono">{code}</span>
        </Badge>
      );
    },
  },
  {
    accessorKey: "duration",
    size: 115,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Duration" />
    ),
    cell: ({ row }) => `${Number(row.getValue("duration"))}ms`,
  },
  {
    accessorKey: "samples",
    size: 115,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Samples" />
    ),
    cell: ({ row }) => {
      const v = Number(row.getValue("samples"));
      return (
        <span className="tabular-nums">
          {Number.isFinite(v) ? formatUnit(v) : "-"}
        </span>
      );
    },
  },
  {
    accessorKey: "timeRange",
    size: 135,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Time Range" />
    ),
    cell: ({ row }) => {
      const type = String(row.original.type);
      const start = row.original.start;
      const end = row.original.end;

      const renderTooltip = (
        content: React.ReactNode,
        tooltipContent: React.ReactNode,
      ) => (
        <Tooltip>
          <TooltipTrigger asChild>{content}</TooltipTrigger>
          <TooltipContent
            side="top"
            className="bg-slate-900 text-white border-slate-700 px-3 py-2"
          >
            {tooltipContent}
          </TooltipContent>
        </Tooltip>
      );

      // Show "-" for instant queries with tooltip showing the start time
      if (type === "instant") {
        if (start) {
          try {
            const formattedStart = formatUTCtoLocal(
              start,
              "dd/MM/yyyy HH:mm:ss",
            );
            return renderTooltip(
              <span className="text-muted-foreground cursor-help">-</span>,
              <span className="font-mono text-xs">{formattedStart}</span>,
            );
          } catch {
            return <span className="text-muted-foreground">-</span>;
          }
        }
        return <span className="text-muted-foreground">-</span>;
      }

      // Show duration for range queries if we have both start and end
      if (start && end) {
        try {
          const formattedStart = formatUTCtoLocal(start, "dd/MM/yyyy HH:mm:ss");
          const formattedEnd = formatUTCtoLocal(end, "dd/MM/yyyy HH:mm:ss");
          const duration = formatTimeRangeDuration(start, end);

          return renderTooltip(
            <span className="font-medium cursor-help">{duration}</span>,
            <div className="font-mono text-xs">
              <div>{formattedStart}</div>
              <div className="text-slate-400">to</div>
              <div>{formattedEnd}</div>
            </div>,
          );
        } catch {
          return (
            <span className="font-medium">
              {formatTimeRangeDuration(start, end)}
            </span>
          );
        }
      }

      return <span className="text-muted-foreground">-</span>;
    },
  },
  {
    accessorKey: "steps",
    size: 95,
    header: ({ column }) => (
      <DataTableColumnHeader column={column} title="Steps" />
    ),
    cell: ({ row }) => String(row.getValue("steps")),
  },
  {
    // No fixed width: takes what the other columns leave.
    id: "origin",
    header: "Origin",
    cell: ({ row }) => <ExecutionOrigin headers={row.original.httpHeaders} />,
  },
  {
    id: "headers",
    size: 115,
    header: "Headers",
    cell: ({ row }) => (
      <HeadersPopover
        headers={row.original.httpHeaders}
        open={headers.isOpen(row.index)}
        onOpenChange={(open) => headers.setOpen(row.index, open)}
      />
    ),
  },
];

const PAGE_KEY = "queryExecutionsPage";
const PAGE_SIZE_KEY = "queryExecutionsPageSize";
const SORTING_KEY = "queryExecutionsSorting";

interface Props {
  fingerprint?: string;
}

export const QueryExecutions: React.FC<Props> = ({ fingerprint }) => {
  const { dateRange } = useDateRange();
  const fromISO = dateRange?.from?.toISOString();
  const toISO = dateRange?.to?.toISOString();

  const [searchParams, setSearchParams] = useSearchParams();
  const page: number = Number(searchParams.get(PAGE_KEY)) || 1;
  const pageSize: number = Number(searchParams.get(PAGE_SIZE_KEY)) || 10;
  const sorting: SortingState = useMemo(
    () =>
      searchParams.get(SORTING_KEY)
        ? (JSON.parse(searchParams.get(SORTING_KEY) as string) as SortingState)
        : [
            {
              id: "timestamp",
              desc: true,
            },
          ],
    [searchParams],
  );

  const handleSortingChange = (newSorting: SortingState) => {
    setSearchParams((prev) => {
      prev.set(SORTING_KEY, JSON.stringify(newSorting));
      return prev;
    });
  };

  const handlePageChange = (newPage: number) => {
    setSearchParams((prev) => {
      prev.set(PAGE_KEY, newPage.toString());
      return prev;
    });
  };

  const handlePageSizeChange = (size: number) => {
    setSearchParams((prev) => {
      prev.set(PAGE_SIZE_KEY, size.toString());
      prev.delete(PAGE_KEY);
      return prev;
    });
  };

  const sortBy = useMemo(() => sorting[0]?.id || "timestamp", [sorting]);
  const serverSortBy = useMemo(() => {
    switch (sortBy) {
      case "timestamp":
        return "ts";
      case "status":
        return "statusCode";
      default:
        return sortBy;
    }
  }, [sortBy]);
  const sortOrder = useMemo(
    () => (sorting[0]?.desc ? "desc" : "asc"),
    [sorting],
  );

  const filters = useExecutionsFilters();
  const [openHeaders, setOpenHeaders] = useState<number | null>(null);
  const columns = useMemo(
    () =>
      buildColumns({
        isOpen: (row) => openHeaders === row,
        setOpen: (row, open) => setOpenHeaders(open ? row : null),
      }),
    [openHeaders],
  );

  const { data, isLoading } = useQuery<PagedResult<QueryExecution>>({
    queryKey: [
      "queryExecutions",
      filters.type,
      filters.status,
      filters.origin,
      filters.debouncedHeaderSearch,
      fingerprint,
      fromISO,
      toISO,
      page,
      pageSize,
      serverSortBy,
      sortOrder,
    ],
    queryFn: () =>
      getQueryExecutions(
        fingerprint || "",
        fromISO,
        toISO,
        page,
        pageSize,
        serverSortBy,
        sortOrder,
        filters.type,
        filters.status,
        filters.origin,
        filters.debouncedHeaderSearch,
      ),
    enabled: Boolean(fingerprint),
  });

  return (
    <div className="space-y-3">
      <ExecutionsFilters filters={filters} />
      {isLoading ? (
        <div className="p-2 text-sm text-muted-foreground">Loading...</div>
      ) : (
        <DataTable<QueryExecution>
          data={data?.data || []}
          columns={columns}
          pagination={true}
          pageSize={pageSize}
          className="w-full"
          serverSide={true}
          sortingState={sorting}
          currentPage={page}
          totalPages={data?.totalPages || 1}
          onSortingChange={(s) => handleSortingChange(s)}
          onPaginationChange={(p) => handlePageChange(p)}
          onPageSizeChange={handlePageSizeChange}
        />
      )}
    </div>
  );
};
