# Non-compliant: connections are auto-accepted without manual approval.
resource "aws_vpc_endpoint_service" "fail_example" {
  acceptance_required        = false
  network_load_balancer_arns = [aws_lb.example.arn]
}

resource "aws_lb" "example" {
  name               = "example-nlb"
  load_balancer_type = "network"
  subnets            = ["subnet-12345678"]
}
