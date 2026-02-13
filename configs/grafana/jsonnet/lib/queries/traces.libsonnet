local c = import 'common.libsonnet';

local selector = '{service_name="%s"}' % c.serviceName;

{
  traceCorrelatedLogRate: 'sum(rate(%s |= "trace_id=" [5m]))' % selector,
  authRequestP95: 'histogram_quantile(0.95, sum(rate(auth_request_duration_seconds_bucket{job="%s"}[5m])) by (le, endpoint))' % c.job,
  traceEventsByType: 'sum by (event) (rate(%s |= "trace_id=" | json | event!="" [5m]))' % selector,
  recentTraceCorrelatedLogs: '%s |= "trace_id="' % selector,

  uniqueTraceIDsLastHour: 'count(count by (trace_id) (count_over_time(%s | json | trace_id!="" [1h])))' % selector,
  traceErrorLogRate: 'sum(rate(%s | json | trace_id!="" | level="error" [5m]))' % selector,
  traceWarnLogRate: 'sum(rate(%s | json | trace_id!="" | level="warn" [5m]))' % selector,
  traceLinkedRequests: 'sum(rate(%s | json | trace_id!="" | message="Request completed" [5m]))' % selector,
}
