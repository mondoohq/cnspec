# Non-compliant: health check uses plaintext HTTP_STR_MATCH.
resource "aws_route53_health_check" "fail_str_match" {
  fqdn              = "example.com"
  port              = 80
  type              = "HTTP_STR_MATCH"
  resource_path     = "/health"
  search_string     = "OK"
  failure_threshold = 3
  request_interval  = 30
}
