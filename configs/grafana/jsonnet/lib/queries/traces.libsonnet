local c = import 'common.libsonnet';

{
  traceCorrelatedLogRate: 'sum(rate({service_name="%s"} |= "trace_id=" [5m]))' % c.serviceName,
  authRequestP95: 'histogram_quantile(0.95, sum(rate(auth_request_duration_seconds_bucket{job="%s"}[5m])) by (le))' % c.job,
  traceEventsByType: 'sum by (event) (rate({service_name="%s"} |= "trace_id=" | json | event!="" [5m]))' % c.serviceName,
  recentTraceCorrelatedLogs: '{service_name="%s"} |= "trace_id="' % c.serviceName,
}
