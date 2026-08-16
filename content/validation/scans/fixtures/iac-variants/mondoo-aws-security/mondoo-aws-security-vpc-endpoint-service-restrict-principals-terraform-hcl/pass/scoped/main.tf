resource "aws_vpc_endpoint_service" "svc" {
  acceptance_required        = true
  network_load_balancer_arns = ["arn:aws:elasticloadbalancing:us-east-1:123456789012:loadbalancer/net/svc/abc123"]
  allowed_principals         = ["arn:aws:iam::123456789012:root"]
}
