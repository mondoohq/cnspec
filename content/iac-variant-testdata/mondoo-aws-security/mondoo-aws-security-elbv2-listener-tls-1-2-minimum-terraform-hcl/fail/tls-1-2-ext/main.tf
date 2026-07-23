# Non-compliant: HTTPS listener uses ELBSecurityPolicy-TLS-1-2-Ext-2018-06,
# which the policy explicitly excludes for enabling weaker cipher suites.
resource "aws_lb_listener" "fail_example" {
  load_balancer_arn = "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/example/abc"
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS-1-2-Ext-2018-06"
}
