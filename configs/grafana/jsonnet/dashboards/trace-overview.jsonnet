local dashboard = import '../lib/dashboard.libsonnet';
local panels = import '../lib/panels.libsonnet';
local q = import '../lib/queries/traces.libsonnet';

local panelList = [
  panels.stat(1, 'Trace-Correlated Log Rate', 0, 0, 12, 8, 'loki', 'loki', q.traceCorrelatedLogRate, 'reqps', null, 'range'),
  panels.stat(2, 'Auth Request p95 (Exemplar-enabled)', 12, 0, 12, 8, 'prometheus', 'mimir', q.authRequestP95, 's', 'p95'),
  panels.timeseries(3, 'Trace-Correlated Events by Type', 0, 8, 24, 9, 'loki', 'loki', q.traceEventsByType, '{{event}}', null, null, null, 'range'),
  panels.logs(4, 'Recent Trace-Correlated Logs', 0, 17, 24, 10, q.recentTraceCorrelatedLogs, 'range'),
];

dashboard.new('Trace Overview', 'trace-overview', 2, ['otel', 'traces', 'correlation'], panelList)
