resource "aws_transfer_server" "example" {
  endpoint_type          = "VPC"
  identity_provider_type = "SERVICE_MANAGED"

  endpoint_details {
    vpc_id     = aws_vpc.main.id
    subnet_ids = [aws_subnet.main.id]
  }
}
