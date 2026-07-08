# Non-compliant: listener uses plaintext HTTP protocol.
resource "aws_lb_listener" "fail_example" {
  load_balancer_arn = "arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/app/example/abc"
  port              = 80
  protocol          = "HTTP"
}
