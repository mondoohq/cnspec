# Compliant: ELBSecurityPolicy-TLS-1-2-Ext-2018-06 negotiates TLS 1.2 only.
resource "aws_lb_listener" "pass_example" {
  load_balancer_arn = "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/example/abc"
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-TLS-1-2-Ext-2018-06"
}
