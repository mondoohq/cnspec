# Compliant: endpoint service requires manual acceptance of connections.
resource "aws_vpc_endpoint_service" "pass_example" {
  acceptance_required        = true
  network_load_balancer_arns = [aws_lb.example.arn]
}

resource "aws_lb" "example" {
  name               = "example-nlb"
  load_balancer_type = "network"
  subnets            = ["subnet-12345678"]
}
