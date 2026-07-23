# Compliant: HTTPS listener uses a forward-secrecy TLS 1.2 security policy.
resource "aws_lb_listener" "pass_example" {
  load_balancer_arn = "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/example/abc"
  port              = 443
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-FS-1-2-Res-2020-10"
}
