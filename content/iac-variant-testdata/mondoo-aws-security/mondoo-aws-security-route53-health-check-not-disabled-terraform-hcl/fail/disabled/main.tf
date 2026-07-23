# Non-compliant: health check is disabled.
resource "aws_route53_health_check" "fail_example" {
  fqdn              = "example.com"
  port              = 443
  type              = "HTTPS"
  failure_threshold = 3
  request_interval  = 30
  disabled          = true
}
