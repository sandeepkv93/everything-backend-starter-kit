local dashboard = import '../lib/dashboard.libsonnet';
local panels = import '../lib/panels.libsonnet';
local q = import '../lib/queries/logs.libsonnet';

local panelList = [
  panels.logs(1, 'Application Logs', 0, 0, 24, 9, q.appLogs),
];

dashboard.new('Logs Overview', 'logs-overview', 1, ['otel', 'logs'], panelList)
