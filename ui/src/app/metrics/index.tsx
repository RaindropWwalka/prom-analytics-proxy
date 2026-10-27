import { useState } from "react";
import { MetricsExplorerHeader } from "@/components/metrics-explorer/metrics-explorer-header";
import { MetricsTable } from "@/components/metrics-explorer/metrics-table";
import { useSeriesMetadataTable } from "./use-metrics-data";
import { TableState } from "@/lib/types";
import { LoadingState } from "./loading";
import { useDebounce } from "@/hooks/use-debounce";

// initialFilters reads filters a link can open the catalog with, as the
// unused-metrics analysis does: ?usage=unused&job=<job>&service=<service>&source=<source>.
function initialFilters(): {
  usage: "all" | "used" | "unused";
  job: string;
  service: string;
  source: string;
} {
  const params = new URLSearchParams(window.location.search);
  const usage = params.get("usage");
  return {
    usage: usage === "used" || usage === "unused" ? usage : "all",
    job: params.get("job") ?? "",
    service: params.get("service") ?? "",
    source: params.get("source") ?? "",
  };
}

export default function MetricsExplorer() {
  const [initial] = useState(initialFilters);
  const [searchQuery, setSearchQuery] = useState("");
  const [typeFilter, setTypeFilter] = useState("all");
  const [usageFilter, setUsageFilter] = useState<"all" | "used" | "unused">(
    initial.usage,
  );
  const [producerFilter, setProducerFilter] = useState<string>(initial.job);
  const [sourceFilter, setSourceFilter] = useState<string>(initial.source);
  const [serviceFilter, setServiceFilter] = useState<string>(initial.service);
  const [tableState, setTableState] = useState<TableState>({
    page: 1,
    pageSize: 10,
    sortBy: "queryCount",
    sortOrder: "desc",
    filter: "",
    type: "all",
  });

  // Increase debounce delay to 750ms for better performance
  const debouncedSearchQuery = useDebounce(searchQuery, 750);

  const { data, isLoading, error } = useSeriesMetadataTable(
    tableState,
    debouncedSearchQuery,
    usageFilter,
    producerFilter,
    sourceFilter,
    serviceFilter,
  );

  if (isLoading) {
    return <LoadingState />;
  }

  if (error) {
    return <div>Error: {error.message}</div>;
  }

  const handleTypeFilterChange = (value: string) => {
    setTypeFilter(value);
    setTableState((prev) => ({
      ...prev,
      page: 1, // Reset to first page when changing type
      type: value,
    }));
  };

  return (
    <div className="p-4">
      <MetricsExplorerHeader
        searchQuery={searchQuery}
        onSearchChange={setSearchQuery}
        typeFilter={typeFilter}
        onTypeFilterChange={handleTypeFilterChange}
        usageFilter={usageFilter}
        onUsageFilterChange={setUsageFilter}
        producers={data.producers}
        producerFilter={producerFilter}
        onProducerFilterChange={setProducerFilter}
        sources={data.sources}
        sourceFilter={sourceFilter}
        onSourceFilterChange={setSourceFilter}
        services={data.services}
        serviceFilter={serviceFilter}
        onServiceFilterChange={setServiceFilter}
      />
      <MetricsTable
        metrics={data.metrics}
        searchQuery={debouncedSearchQuery}
        tableState={tableState}
        onTableStateChange={setTableState}
      />
    </div>
  );
}
