import { Copy } from "lucide-react";
import { Button } from "../ui/button";
import { toast } from "sonner";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../ui/tabs";
import { KeyMetrics } from "../key-metrics";
import QueryTimeRangeDistribution from "../query-time-range-distribution";
import { QueryLatencyTrends } from "../query-latency-trends";
import { QueryExecutions } from "./table";
import { StatusBreakdown } from "../status-breakdown";
import { useSearchState } from "@/hooks/use-search-state";

const TAB_KEY = "queryDetailsTab";

interface QueryDetailsProps {
  onClose: () => void;
  query: string;
  fingerprint?: string;
}

export function QueryDetails({ query, fingerprint }: QueryDetailsProps) {
  const [tab, setTab] = useSearchState(TAB_KEY, "overview");

  const copyToClipboard = () => {
    navigator.clipboard.writeText(query);
    toast.success("Query copied to clipboard");
  };

  const handleTabChange = (value: string) => {
    setTab(value);
  };

  return (
    // Fills the sheet below its header and scrolls on its own: a fixed max
    // height left the sheet's bottom blank and cut the charts off.
    <div className="min-h-0 flex-1 overflow-y-auto px-4 pb-6">
      <div className="mb-6 p-3 bg-foreground/2 rounded-lg border flex items-center justify-between gap-3">
        <code className="text-sm font-mono flex-1 break-all">{query}</code>
        <Button
          variant="ghost"
          size="sm"
          onClick={copyToClipboard}
          className="shrink-0 cursor-pointer"
        >
          <Copy className="h-4 w-4" />
        </Button>
      </div>
      <Tabs value={tab} onValueChange={handleTabChange}>
        <TabsList className="flex bg-muted rounded-lg overflow-hidden w-full grid-cols-3">
          <TabsTrigger value="overview" className="flex-1 py-3 px-5">
            Overview
          </TabsTrigger>
          <TabsTrigger value="executions" className="flex-1 py-3 px-5">
            Executions
          </TabsTrigger>
        </TabsList>
        <TabsContent value="overview" className="mt-2 grid gap-4">
          <KeyMetrics fingerprint={fingerprint} />
          <QueryTimeRangeDistribution fingerprint={fingerprint} />
          <div className="grid gap-4 lg:grid-cols-2">
            <QueryLatencyTrends fingerprint={fingerprint} />
            <StatusBreakdown fingerprint={fingerprint} />
          </div>
        </TabsContent>
        <TabsContent value="executions" className="mt-2">
          <QueryExecutions fingerprint={fingerprint} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
