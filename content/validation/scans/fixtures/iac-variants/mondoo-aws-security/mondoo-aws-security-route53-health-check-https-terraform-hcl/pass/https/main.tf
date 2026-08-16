# Compliant: health check uses HTTPS, not HTTP/HTTP_STR_MATCH/TCP.
resource "aws_route53_health_check" "pass_example" {
  fqdn              = "example.com"
  port              = 443
  type              = "HTTPS"
  resource_path     = "/health"
  failure_threshold = 3
  request_interval  = 30
}
