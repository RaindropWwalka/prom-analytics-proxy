import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Filter } from "lucide-react";

interface MetricsExplorerHeaderProps {
  searchQuery: string;
  onSearchChange: (value: string) => void;
  typeFilter: string;
  onTypeFilterChange: (value: string) => void;
  usageFilter?: "all" | "used" | "unused";
  onUsageFilterChange?: (value: "all" | "used" | "unused") => void;
  producers?: string[];
  producerFilter?: string;
  onProducerFilterChange?: (value: string) => void;
  sources?: string[];
  sourceFilter?: string;
  onSourceFilterChange?: (value: string) => void;
  services?: string[];
  serviceFilter?: string;
  onServiceFilterChange?: (value: string) => void;
}

export function MetricsExplorerHeader({
  searchQuery,
  onSearchChange,
  typeFilter,
  onTypeFilterChange,
  usageFilter = "all",
  onUsageFilterChange,
  producers = [],
  producerFilter = "",
  onProducerFilterChange,
  sources = [],
  sourceFilter = "",
  onSourceFilterChange,
  services = [],
  serviceFilter = "",
  onServiceFilterChange,
}: MetricsExplorerHeaderProps) {
  const producerOptions = producers.filter((p) => !!p && p.trim().length > 0);
  const currentProducerValue = producerOptions.includes(producerFilter)
    ? producerFilter
    : "__all__";
  return (
    <div className="flex flex-col gap-2">
      <div>
        <h1 className="text-2xl font-bold">Metrics Catalog</h1>
        <p className="text-sm text-muted-foreground">
          Browse and analyze patterns and usage of your metrics
        </p>
      </div>
      <div className="flex flex-col gap-2 lg:flex-row lg:items-center">
        <Input
          placeholder="Search queries..."
          className="lg:min-w-[140px] lg:max-w-[300px] lg:flex-[2]"
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
        />
        <Select value={typeFilter} onValueChange={onTypeFilterChange}>
          <SelectTrigger className="w-full lg:min-w-[110px] lg:max-w-[150px] lg:flex-1">
            <div className="flex min-w-0 items-center gap-2">
              <Filter className="h-4 w-4" />
              <SelectValue placeholder="All Types" />
            </div>
          </SelectTrigger>
          <SelectContent className="max-w-[90vw] sm:max-w-[600px] whitespace-normal break-words">
            <SelectItem value="all">All Types</SelectItem>
            <SelectItem value="counter">Counter</SelectItem>
            <SelectItem value="gauge">Gauge</SelectItem>
            <SelectItem value="histogram">Histogram</SelectItem>
            <SelectItem value="summary">Summary</SelectItem>
          </SelectContent>
        </Select>
        <Select
          value={currentProducerValue}
          onValueChange={(v) =>
            onProducerFilterChange?.(v === "__all__" ? "" : v)
          }
        >
          <SelectTrigger className="w-full text-left lg:min-w-[130px] lg:max-w-[320px] lg:flex-[2]">
            <div className="flex items-center gap-2 min-w-0">
              <Filter className="h-4 w-4 shrink-0" />
              <span
                className="truncate"
                title={producerFilter || "All Producers"}
              >
                <SelectValue placeholder="All Producers" />
              </span>
            </div>
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="__all__">All Producers</SelectItem>
            {producerOptions.map((producer) => (
              <SelectItem key={producer} value={producer}>
                {producer}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {services.length > 0 && (
          <Select
            value={services.includes(serviceFilter) ? serviceFilter : "__all__"}
            onValueChange={(v) =>
              onServiceFilterChange?.(v === "__all__" ? "" : v)
            }
          >
            <SelectTrigger
              className="w-full text-left lg:min-w-[130px] lg:max-w-[260px] lg:flex-[2]"
              aria-label="Service"
            >
              <div className="flex items-center gap-2 min-w-0">
                <Filter className="h-4 w-4 shrink-0" />
                <span
                  className="truncate"
                  title={serviceFilter || "All Services"}
                >
                  <SelectValue placeholder="All Services" />
                </span>
              </div>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__all__">All Services</SelectItem>
              {services.map((service) => (
                <SelectItem key={service} value={service}>
                  {service}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
        {sources.length > 0 && (
          <Select
            value={sources.includes(sourceFilter) ? sourceFilter : "__all__"}
            onValueChange={(v) =>
              onSourceFilterChange?.(v === "__all__" ? "" : v)
            }
          >
            <SelectTrigger
              className="w-full text-left lg:min-w-[130px] lg:max-w-[260px] lg:flex-[2]"
              aria-label="Source"
            >
              <div className="flex items-center gap-2 min-w-0">
                <Filter className="h-4 w-4 shrink-0" />
                <span
                  className="truncate"
                  title={sourceFilter || "All Sources"}
                >
                  <SelectValue placeholder="All Sources" />
                </span>
              </div>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="__all__">All Sources</SelectItem>
              {sources.map((source) => (
                <SelectItem key={source} value={source}>
                  {source}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}
        <Select
          value={usageFilter}
          onValueChange={(v) =>
            onUsageFilterChange?.(v as "all" | "used" | "unused")
          }
        >
          <SelectTrigger className="w-full lg:min-w-[120px] lg:max-w-[170px] lg:flex-1">
            <div className="flex min-w-0 items-center gap-2">
              <Filter className="h-4 w-4" />
              <SelectValue placeholder="All Metrics" />
            </div>
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Metrics</SelectItem>
            <SelectItem value="used">Used Only</SelectItem>
            <SelectItem value="unused">Unused Only</SelectItem>
          </SelectContent>
        </Select>
      </div>
    </div>
  );
}
