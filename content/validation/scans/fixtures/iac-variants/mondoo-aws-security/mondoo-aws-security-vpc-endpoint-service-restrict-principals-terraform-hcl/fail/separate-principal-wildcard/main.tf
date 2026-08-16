resource "aws_vpc_endpoint_service" "this" {
  acceptance_required        = true
  network_load_balancer_arns = ["arn:aws:elasticloadbalancing:us-east-1:111122223333:loadbalancer/net/example/50dc6c495c0c9188"]
}

resource "aws_vpc_endpoint_service_allowed_principal" "this" {
  vpc_endpoint_service_id = aws_vpc_endpoint_service.this.id
  principal_arn           = "*"
}
