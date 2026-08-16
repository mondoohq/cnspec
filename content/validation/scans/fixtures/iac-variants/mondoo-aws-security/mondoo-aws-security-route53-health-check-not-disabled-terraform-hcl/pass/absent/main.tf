# Compliant: disabled attribute omitted, defaults to false (health check active).
resource "aws_route53_health_check" "pass_absent" {
  fqdn              = "example.com"
  port              = 443
  type              = "HTTPS"
  failure_threshold = 3
  request_interval  = 30
}
