package db

import (
	"time"
)

type QueryType string
type DatabaseProvider string

const (
	QueryTypeRange   QueryType        = "range"
	QueryTypeInstant QueryType        = "instant"
	PostGreSQL       DatabaseProvider = "postgresql"
	SQLite           DatabaseProvider = "sqlite"
)

var (
	KnownQueryTypes = []QueryType{QueryTypeRange, QueryTypeInstant}
)

type LabelMatchers []map[string]string

type Query struct {
	TS                    time.Time         `json:"-"`
	QueryParam            string            `json:"query_param"`
	TimeParam             time.Time         `json:"time_param"`
	Duration              time.Duration     `json:"duration"`
	StatusCode            int               `json:"status_code"`
	BodySize              int               `json:"body_size"`
	LabelMatchers         LabelMatchers     `json:"-"`
	Fingerprint           string            `json:"-"`
	Type                  QueryType         `json:"type"`
	Step                  float64           `json:"step"`
	Start                 time.Time         `json:"start"`
	End                   time.Time         `json:"end"`
	TotalQueryableSamples int               `json:"total_queryable_samples"`
	PeakSamples           int               `json:"peak_samples"`
	HTTPHeaders           map[string]string `json:"http_headers,omitempty"`
}

type QueryTypesResult struct {
	TotalQueries   *int     `json:"total_queries"`
	InstantPercent *float64 `json:"instant_percent"`
	RangePercent   *float64 `json:"range_percent"`
}

type AverageDurationResult struct {
	AvgDuration  *float64 `json:"avg_duration"`
	DeltaPercent *float64 `json:"delta_percent"`
}

type QueryRateResult struct {
	SuccessTotal       *int     `json:"success_total"`
	SuccessRatePercent *float64 `json:"success_rate_percent"`
	ErrorTotal         *int     `json:"error_total"`
	ErrorRatePercent   *float64 `json:"error_rate_percent"`
}

type PagedResult struct {
	TotalPages int         `json:"totalPages"`
	Total      int         `json:"total"`
	Data       interface{} `json:"data"`
}

type QueriesBySerieNameResult struct {
	Query           string            `json:"query"`
	AvgDuration     float64           `json:"avgDuration"`
	AvgPeakySamples float64           `json:"avgPeakySamples"`
	MaxPeakSamples  int               `json:"maxPeakSamples"`
	HTTPHeaders     map[string]string `json:"httpHeaders"`
	// TS is when the query last ran in the range.
	TS time.Time `json:"ts"`
	// Executions splits into DashboardRuns (a Grafana panel), AlertRuns
	// (Grafana alerting) and the rest, by the headers stored with each run.
	Executions    int `json:"executions"`
	DashboardRuns int `json:"dashboardRuns"`
	AlertRuns     int `json:"alertRuns"`
	// Which dashboards (by title, else uid), alert rules and other clients
	// (X-Client-Name, else User-Agent) ran it.
	Dashboards []string `json:"dashboards,omitempty"`
	Rules      []string `json:"rules,omitempty"`
	Clients    []string `json:"clients,omitempty"`
}

type QueryStatusDistributionResult struct {
	// Time is the bucket's start, in UTC.
	Time      time.Time `json:"time"`
	Status2xx int       `json:"2xx"`
	Status4xx int       `json:"4xx"`
	Status5xx int       `json:"5xx"`
}

type QueryLatencyTrendsResult struct {
	Value float64 `json:"value"`
	P95   int     `json:"p95"`
	// Time is the bucket's start, in UTC.
	Time time.Time `json:"time"`
}

type QueryThroughputAnalysisResult struct {
	Value float64 `json:"value"`
	// Time is the bucket's start, in UTC.
	Time time.Time `json:"time"`
}

type QueryErrorAnalysisResult struct {
	// Time is the bucket's start, in UTC.
	Time  time.Time `json:"time"`
	Value float64   `json:"value"`
}

// QueryTimeRangeDistributionResult represents a bucketed count of range query window sizes.
// Label corresponds to human-friendly bucket labels like "<24h", "24h", "7d", "30d", "60d", "90d+".
type QueryTimeRangeDistributionResult struct {
	Label   string  `json:"label"`
	Count   int     `json:"count"`
	Percent float64 `json:"percent"`
}

type MetricProducersResult struct {
	Job string `json:"job"`
	// Source is the Prometheus the series came from, as told by the
	// external label named by producers_source_label; empty without one.
	Source string `json:"source,omitempty"`
	// Service is the series' producers_service_label, when it has one: a
	// job such as a shared JMX exporter can carry many services.
	Service string `json:"service,omitempty"`
	Series  int    `json:"series"`
}

type MetricUsageStatics struct {
	SerieCount      int                     `json:"serieCount"`
	LabelCount      int                     `json:"labelCount"`
	AlertCount      int                     `json:"alertCount"`
	RecordCount     int                     `json:"recordCount"`
	DashboardCount  int                     `json:"dashboardCount"`
	TotalAlerts     int                     `json:"totalAlerts"`
	TotalRecords    int                     `json:"totalRecords"`
	TotalDashboards int                     `json:"totalDashboards"`
	Producers       []MetricProducersResult `json:"producers"`
}

type RuleUsageKind string

const (
	RuleUsageKindAlert  RuleUsageKind = "alert"
	RuleUsageKindRecord RuleUsageKind = "record"
)

type RulesUsage struct {
	Serie      string   `json:"serie"`
	GroupName  string   `json:"group_name"`
	Name       string   `json:"name"`
	Expression string   `json:"expression"`
	Kind       string   `json:"kind"`
	Labels     []string `json:"labels"`
	// PromLink points at the rule in the Prometheus evaluating it.
	PromLink   string    `json:"prom_link,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
}

type DashboardUsage struct {
	Id        string    `json:"id"`
	Serie     string    `json:"serie"`
	Name      string    `json:"title"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

type RulesUsageParams struct {
	Serie     string
	Kind      string
	Filter    string
	Page      int
	PageSize  int
	SortBy    string
	SortOrder string
	TimeRange TimeRange
}

type DashboardUsageParams struct {
	Serie     string
	Filter    string
	Page      int
	PageSize  int
	SortBy    string
	SortOrder string
	TimeRange TimeRange
}

type MetricQueryPerformanceStatistics struct {
	QueryRate       *QueryRateResult `json:"queryRate"`
	TotalQueries    *int             `json:"totalQueries"`
	AverageSamples  *float64         `json:"averageSamples"`
	PeakSamples     *int             `json:"peakSamples"`
	AverageDuration *float64         `json:"averageDuration"`
	SampleRate      *float64         `json:"sampleRate"`
}

// QueryExpression aggregates query metrics grouped by fingerprint
type QueryExpression struct {
	Fingerprint      string  `json:"fingerprint"`
	Query            string  `json:"query"`
	Executions       int     `json:"executions"`
	AvgDuration      float64 `json:"avgDuration"`
	ErrorRatePercent float64 `json:"errorRatePercent"`
	PeakSamples      int     `json:"peakSamples"`
}
