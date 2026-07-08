resource "aws_vpc_block_public_access_options" "example" {
  internet_gateway_block_mode = "block-ingress"
}
