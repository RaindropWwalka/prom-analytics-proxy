import { useQuery } from "@tanstack/react-query";
import {
  getMetricQueryPerformanceStatistics,
  getMetricMetadata,
  getMetricStatistics,
  getSeriesMetadata,
  getSerieExpressions,
  getMetricUsage,
  getProducers,
  getSources,
  getServices,
  type MetricUsageResponse,
  type UsageQuery,
} from "@/api/metrics";
import {
  PagedResult,
  TableState,
  MetricMetadata,
  MetricStatistics,
  MetricQueryPerformanceStatistics,
  QueryLatencyTrendsResult,
  DateRange,
} from "@/lib/types";
import { getQueryLatencyTrends } from "@/api/queries";

interface MetricsData {
  metrics: PagedResult<MetricMetadata> | undefined;
  producers: string[] | undefined;
  sources: string[] | undefined;
  services: string[] | undefined;
}

interface MetricStatisticsData {
  statistics: MetricStatistics | undefined;
}

export function useSeriesMetadataTable(
  tableState?: TableState,
  searchQuery?: string,
  usage?: "all" | "used" | "unused",
  job?: string,
  source?: string,
  service?: string,
) {
  const {
    data: metrics,
    isLoading,
    error,
  } = useQuery<PagedResult<MetricMetadata>>({
    queryKey: ["metrics", tableState, searchQuery, usage, job, source, service],
    queryFn: () =>
      getSeriesMetadata(
        tableState?.page || 1,
        tableState?.pageSize || 10,
        tableState?.sortBy || "name",
        tableState?.sortOrder || "asc",
        searchQuery || "",
        tableState?.type || "all",
        usage || "all",
        job,
        source,
        service,
      ),
  });

  const { data: producers } = useQuery<string[]>({
    queryKey: ["producers"],
    queryFn: getProducers,
  });

  const { data: sources } = useQuery<string[]>({
    queryKey: ["sources"],
    queryFn: getSources,
  });

  const { data: services } = useQuery<string[]>({
    queryKey: ["services"],
    queryFn: getServices,
  });

  return {
    data: {
      metrics,
      producers,
      sources,
      services,
    } as MetricsData,
    isLoading,
    error,
  };
}

export function useMetricMetadata(metricName: string) {
  return useQuery<MetricMetadata | null>({
    queryKey: ["metricMetadata", metricName],
    queryFn: () => getMetricMetadata(metricName),
  });
}

export function useMetricStatistics(
  metricName: string,
  timeRange: DateRange | undefined,
) {
  const from = timeRange?.from?.toISOString() || "";
  const to = timeRange?.to?.toISOString() || "";

  const {
    data: statistics,
    isLoading,
    error,
  } = useQuery<MetricStatistics>({
    queryKey: ["metricStatistics", metricName, from, to],
    queryFn: () => getMetricStatistics(metricName, from, to),
  });

  return {
    data: {
      statistics,
    } as MetricStatisticsData,
    isLoading,
    error,
  };
}

export function useMetricQueryPerformanceStatistics(
  metricName: string,
  timeRange: DateRange | undefined,
) {
  const from = timeRange?.from?.toISOString() || "";
  const to = timeRange?.to?.toISOString() || "";

  const { data, isLoading, error } = useQuery<MetricQueryPerformanceStatistics>(
    {
      queryKey: ["metricQueryPerformanceStatistics", metricName, from, to],
      queryFn: () => getMetricQueryPerformanceStatistics(metricName, from, to),
    },
  );

  return {
    data,
    isLoading,
    error,
  };
}

export function useQueryLatencyTrends(
  metricName: string,
  timeRange: DateRange | undefined,
) {
  const from = timeRange?.from?.toISOString() || "";
  const to = timeRange?.to?.toISOString() || "";

  const { data, isLoading, error } = useQuery<QueryLatencyTrendsResult[]>({
    queryKey: ["queryLatencyTrends", metricName, from, to],
    queryFn: () => getQueryLatencyTrends(from, to, metricName),
  });

  return {
    data,
    isLoading,
    error,
  };
}

export function useMetricUsage(
  metricName: string,
  kind: string,
  page: number = 1,
  pageSize: number = 10,
  from?: Date,
  to?: Date,
  query: UsageQuery = {},
) {
  const fromParam = from ? from.toISOString() : "";
  const toParam = to ? to.toISOString() : "";

  return useQuery<MetricUsageResponse, Error>({
    queryKey: [
      "metric-usage",
      metricName,
      kind,
      page,
      pageSize,
      fromParam,
      toParam,
      query.sortBy,
      query.sortOrder,
      query.filter,
    ],
    queryFn: () =>
      getMetricUsage(
        metricName,
        kind,
        page,
        pageSize,
        fromParam,
        toParam,
        query,
      ),
    enabled: !!metricName && !!kind,
  });
}

export function useSerieExpressions(
  metricName: string,
  tableState?: TableState,
  timeRange?: DateRange | undefined,
) {
  const from = timeRange?.from?.toISOString() || "";
  const to = timeRange?.to?.toISOString() || "";
  const { data, isLoading, error } = useQuery({
    queryKey: ["serie-expressions", metricName, tableState, from, to],
    queryFn: () =>
      getSerieExpressions(
        metricName,
        tableState?.page || 1,
        tableState?.pageSize || 10,
        from,
        to,
        {
          sortBy: tableState?.sortBy,
          sortOrder: tableState?.sortOrder,
          filter: tableState?.filter,
        },
      ),
    enabled: !!metricName,
  });

  return {
    data,
    isLoading,
    error,
  };
}
