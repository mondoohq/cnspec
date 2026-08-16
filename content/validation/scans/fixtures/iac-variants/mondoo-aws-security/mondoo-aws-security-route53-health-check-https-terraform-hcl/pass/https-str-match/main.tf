# Compliant: health check uses HTTPS_STR_MATCH (still encrypted).
resource "aws_route53_health_check" "pass_str_match" {
  fqdn              = "example.com"
  port              = 443
  type              = "HTTPS_STR_MATCH"
  resource_path     = "/health"
  search_string     = "OK"
  failure_threshold = 3
  request_interval  = 30
}
