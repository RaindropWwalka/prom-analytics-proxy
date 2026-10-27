import { useSearchParams } from "wouter";
import { useDebounce } from "@/hooks/use-debounce";

export const TYPE_KEY = "queryExecutionsType";
export const STATUS_KEY = "queryExecutionsStatus";
export const ORIGIN_KEY = "queryExecutionsOrigin";
export const HEADER_SEARCH_KEY = "queryExecutionsHeaderSearch";
const PAGE_KEY = "queryExecutionsPage";

// The selects use "all" for no filter, which the API reads as "".
export const ALL = "all";

export interface ExecutionsFilterState {
  type: "all" | "instant" | "range";
  status: string;
  origin: string;
  headerSearch: string;
  debouncedHeaderSearch: string;
  set: (key: string, value: string) => void;
}

// useExecutionsFilters keeps the executions filters in the URL, next to the
// table's own page and sorting, so a filtered view can be shared.
export function useExecutionsFilters(): ExecutionsFilterState {
  const [searchParams, setSearchParams] = useSearchParams();
  const headerSearch = searchParams.get(HEADER_SEARCH_KEY) ?? "";
  const debouncedHeaderSearch = useDebounce(headerSearch, 500);
  const type = searchParams.get(TYPE_KEY);

  const set = (key: string, value: string) =>
    setSearchParams((prev) => {
      if (value === "" || value === ALL) {
        prev.delete(key);
      } else {
        prev.set(key, value);
      }
      // A narrower filter can have fewer pages than the one being shown.
      prev.delete(PAGE_KEY);
      return prev;
    });

  return {
    type: type === "instant" || type === "range" ? type : "all",
    status: searchParams.get(STATUS_KEY) ?? "",
    origin: searchParams.get(ORIGIN_KEY) ?? "",
    headerSearch,
    debouncedHeaderSearch,
    set,
  };
}
