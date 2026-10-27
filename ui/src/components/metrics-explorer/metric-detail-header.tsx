"use client";

import { Button } from "@/components/ui/button";
import { ArrowLeft } from "lucide-react";
import { PreservedLink } from "@/components/preserved-link.tsx";
import { MetricTypeTag } from "@/components/metrics-explorer/metric-type-tag";
import { MetricSources } from "@/components/metrics-explorer/metric-sources";
import { Skeleton } from "@/components/ui/skeleton";
import { useMetricMetadata } from "@/app/metrics/use-metrics-data";
import { formatUnit } from "@/lib/utils";
import { totalSeries } from "@/lib/utils/series";

interface MetricDetailHeaderProps {
  metricName: string;
}

export function MetricDetailHeader({ metricName }: MetricDetailHeaderProps) {
  const { data: metadata, isLoading } = useMetricMetadata(metricName);

  return (
    <div className="space-y-6">
      <div className="flex items-start gap-4">
        <Button variant="ghost" size="icon" asChild>
          <PreservedLink href="/metrics-explorer">
            <ArrowLeft className="h-4 w-4" />
            <span className="sr-only">Back</span>
          </PreservedLink>
        </Button>
        <div className="min-w-0 flex-1 space-y-2">
          <h1 className="text-2xl font-bold break-all">{metricName}</h1>
          {isLoading ? (
            <Skeleton className="h-4 w-2/3" />
          ) : metadata ? (
            <>
              <p className="text-sm text-muted-foreground whitespace-pre-wrap break-words">
                {metadata.help || "No description"}
              </p>
              <div className="flex flex-wrap items-center gap-x-6 gap-y-2 text-sm">
                <span className="flex items-center gap-2">
                  <span className="text-muted-foreground">Type</span>
                  <MetricTypeTag type={metadata.type} />
                </span>
                {/* Prometheus metadata's unit; exporters rarely set it. */}
                {metadata.unit && (
                  <span className="flex items-center gap-2">
                    <span className="text-muted-foreground">Unit</span>
                    {metadata.unit}
                  </span>
                )}
                {totalSeries(metadata.series) !== undefined && (
                  <span
                    className="flex items-center gap-2"
                    title={Object.entries(metadata.series ?? {})
                      .map(([source, n]) => `${source}: ${n}`)
                      .join("\n")}
                  >
                    <span className="text-muted-foreground">Series</span>
                    <span className="tabular-nums">
                      {formatUnit(totalSeries(metadata.series) ?? 0)}
                    </span>
                  </span>
                )}
                <span className="flex min-w-0 items-center gap-2">
                  <span className="text-muted-foreground">Sources</span>
                  <MetricSources sources={metadata.sources} max={5} />
                </span>
              </div>
            </>
          ) : (
            <p className="text-sm text-muted-foreground">
              Not in the metrics catalog yet
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
