# Non-compliant: Network Load Balancer listener forwards plaintext TCP.
resource "aws_lb_listener" "fail_example" {
  load_balancer_arn = "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/net/example/abc"
  port              = 80
  protocol          = "TCP"

  default_action {
    type             = "forward"
    target_group_arn = "arn:aws:elasticloadbalancing:us-east-1:123456789012:targetgroup/example/abc"
  }
}
