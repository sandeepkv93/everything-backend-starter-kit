local c = import 'common.libsonnet';

{
  requestRateByStatus: 'sum(rate(http_server_request_duration_seconds_count{job="%s"}[5m])) by (http_response_status_code)' % c.job,
  requestLatencyP95: 'histogram_quantile(0.95, sum(rate(http_server_request_duration_seconds_bucket{job="%s"}[5m])) by (le))' % c.job,
  redisCommandLatencyP95: 'histogram_quantile(0.95, sum(rate(redis_command_duration_seconds_bucket{job="%s"}[5m])) by (le, command))' % c.job,
  redisErrorRate: 'redis_command_error_rate_ratio{job="%s"}' % c.job,
  redisPoolSaturation: 'redis_pool_saturation_ratio{job="%s"}' % c.job,
  redisHitRatio: 'redis_keyspace_hit_ratio_ratio{job="%s"}' % c.job,
}
