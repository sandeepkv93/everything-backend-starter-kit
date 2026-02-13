local dashboard = import '../lib/dashboard.libsonnet';
local panels = import '../lib/panels.libsonnet';
local q = import '../lib/queries/api.libsonnet';

local panelList = [
  panels.timeseries(1, 'Request Rate by Status', 0, 0, 12, 9, 'prometheus', 'mimir', q.requestRateByStatus, '{{http_response_status_code}}'),
  panels.timeseries(2, 'Request Latency p95', 12, 0, 12, 9, 'prometheus', 'mimir', q.requestLatencyP95, 'p95', 's'),
  panels.timeseries(3, 'Redis Command Latency p95', 0, 9, 12, 9, 'prometheus', 'mimir', q.redisCommandLatencyP95, '{{command}}', 's'),
  panels.timeseries(4, 'Redis Command Error Rate', 12, 9, 12, 9, 'prometheus', 'mimir', q.redisErrorRate, 'error_rate', 'percentunit', 0, 1),
  panels.timeseries(5, 'Redis Pool Saturation', 0, 18, 12, 9, 'prometheus', 'mimir', q.redisPoolSaturation, 'pool_saturation', 'percentunit', 0, 1),
  panels.timeseries(6, 'Redis Keyspace Hit Ratio', 12, 18, 12, 9, 'prometheus', 'mimir', q.redisHitRatio, 'hit_ratio', 'percentunit', 0, 1),
];

dashboard.new('API Overview', 'api-overview', 2, ['otel', 'api', 'redis'], panelList)
