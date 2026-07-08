# Compliant: health check is not disabled.
resource "aws_route53_health_check" "pass_example" {
  fqdn              = "example.com"
  port              = 443
  type              = "HTTPS"
  failure_threshold = 3
  request_interval  = 30
  disabled          = false
}
