import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  ALL,
  HEADER_SEARCH_KEY,
  ORIGIN_KEY,
  STATUS_KEY,
  TYPE_KEY,
  type ExecutionsFilterState,
} from "./use-executions-filters";

export function ExecutionsFilters({
  filters,
}: {
  filters: ExecutionsFilterState;
}) {
  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
      <Select
        value={filters.type}
        onValueChange={(v) => filters.set(TYPE_KEY, v)}
      >
        <SelectTrigger className="sm:w-[140px]" aria-label="Type">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL}>All types</SelectItem>
          <SelectItem value="instant">Instant</SelectItem>
          <SelectItem value="range">Range</SelectItem>
        </SelectContent>
      </Select>
      <Select
        value={filters.status || ALL}
        onValueChange={(v) => filters.set(STATUS_KEY, v)}
      >
        <SelectTrigger className="sm:w-[150px]" aria-label="Status">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL}>All statuses</SelectItem>
          <SelectItem value="2xx">2xx</SelectItem>
          <SelectItem value="4xx">4xx</SelectItem>
          <SelectItem value="5xx">5xx</SelectItem>
          <SelectItem value="error">Errors (not 2xx)</SelectItem>
        </SelectContent>
      </Select>
      <Select
        value={filters.origin || ALL}
        onValueChange={(v) => filters.set(ORIGIN_KEY, v)}
      >
        <SelectTrigger className="sm:w-[160px]" aria-label="Origin">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value={ALL}>All origins</SelectItem>
          <SelectItem value="dashboard">Dashboards</SelectItem>
          <SelectItem value="alert">Alerts</SelectItem>
          <SelectItem value="other">Other clients</SelectItem>
        </SelectContent>
      </Select>
      <Input
        className="sm:max-w-[280px]"
        placeholder="Search headers (dashboard, rule, curl...)"
        value={filters.headerSearch}
        onChange={(e) => filters.set(HEADER_SEARCH_KEY, e.target.value)}
      />
    </div>
  );
}
