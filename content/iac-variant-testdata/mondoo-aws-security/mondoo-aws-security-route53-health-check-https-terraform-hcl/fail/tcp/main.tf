# Non-compliant: health check uses raw TCP (no TLS).
resource "aws_route53_health_check" "fail_tcp" {
  ip_address        = "192.0.2.10"
  port              = 6379
  type              = "TCP"
  failure_threshold = 3
  request_interval  = 30
}
