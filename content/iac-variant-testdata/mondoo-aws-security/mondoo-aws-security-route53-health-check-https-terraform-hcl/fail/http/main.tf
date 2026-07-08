# Non-compliant: health check uses plaintext HTTP.
resource "aws_route53_health_check" "fail_example" {
  fqdn              = "example.com"
  port              = 80
  type              = "HTTP"
  resource_path     = "/health"
  failure_threshold = 3
  request_interval  = 30
}
