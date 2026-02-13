local dashboard = import '../lib/dashboard.libsonnet';
local panels = import '../lib/panels.libsonnet';
local q = import '../lib/queries/api.libsonnet';

local panelList = [
  panels.stat(1, 'Active Requests', 0, 0, 8, 6, 'prometheus', 'mimir', q.activeRequests, 'short'),
  panels.stat(2, 'Request Latency p95', 8, 0, 8, 6, 'prometheus', 'mimir', q.requestLatencyP95, 's', 'p95'),
  panels.stat(3, 'Request Latency p99', 16, 0, 8, 6, 'prometheus', 'mimir', q.requestLatencyP99, 's', 'p99'),

  panels.timeseriesTargets(4, 'HTTP Status Class Throughput', 0, 6, 24, 8, 'prometheus', 'mimir', [
    panels.target('A', q.requestClass2xx, '2xx'),
    panels.target('B', q.requestClass4xx, '4xx'),
    panels.target('C', q.requestClass5xx, '5xx'),
  ], 'reqps'),

  panels.timeseries(5, 'Request Rate by Status', 0, 14, 12, 8, 'prometheus', 'mimir', q.requestRateByStatus, '{{http_response_status_code}}', 'reqps'),
  panels.timeseries(6, 'Redis Command Latency p95', 12, 14, 12, 8, 'prometheus', 'mimir', q.redisCommandLatencyP95, '{{command}}', 's'),

  panels.timeseries(7, 'Auth Login Attempts', 0, 22, 8, 8, 'prometheus', 'mimir', q.authLoginAttempts, '{{status}}', 'reqps'),
  panels.timeseries(8, 'Auth Refresh Attempts', 8, 22, 8, 8, 'prometheus', 'mimir', q.authRefreshAttempts, '{{status}}', 'reqps'),
  panels.timeseries(9, 'Rate Limit Decisions', 16, 22, 8, 8, 'prometheus', 'mimir', q.rateLimitDecisions, '{{outcome}}', 'reqps'),

  panels.timeseries(10, 'Repository Errors by Repo', 0, 30, 12, 8, 'prometheus', 'mimir', q.repositoryErrors, '{{repo}}', 'reqps'),
  panels.timeseries(11, 'Unhealthy Health Checks', 12, 30, 12, 8, 'prometheus', 'mimir', q.unhealthyChecks, '{{check}}', 'reqps'),

  panels.timeseries(12, 'Redis Ops by Status', 0, 38, 8, 8, 'prometheus', 'mimir', q.redisOpsByStatus, '{{status}}', 'reqps'),
  panels.timeseries(13, 'Redis Command Error Rate', 8, 38, 8, 8, 'prometheus', 'mimir', q.redisErrorRate, 'error_rate', 'percentunit', 0, 1),
  panels.timeseries(14, 'Redis Pool Saturation', 16, 38, 8, 8, 'prometheus', 'mimir', q.redisPoolSaturation, 'pool_saturation', 'percentunit', 0, 1),

  panels.piechart(15, 'Top Redis Commands', 0, 46, 12, 8, 'prometheus', 'mimir', [
    panels.target('A', q.redisTopCommands, '{{command}}'),
  ]),
  panels.timeseries(16, 'Redis Keyspace Hit Ratio', 12, 46, 12, 8, 'prometheus', 'mimir', q.redisHitRatio, 'hit_ratio', 'percentunit', 0, 1),
];

dashboard.new('API Overview', 'api-overview', 3, ['otel', 'api', 'redis', 'auth'], panelList)
