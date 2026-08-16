# Compliant: gateway load balancer endpoint service requiring acceptance.
resource "aws_vpc_endpoint_service" "pass_example" {
  acceptance_required        = true
  gateway_load_balancer_arns = [aws_lb.example.arn]

  allowed_principals = ["arn:aws:iam::123456789012:root"]
}

resource "aws_lb" "example" {
  name               = "example-gwlb"
  load_balancer_type = "gateway"
  subnets            = ["subnet-12345678"]
}
