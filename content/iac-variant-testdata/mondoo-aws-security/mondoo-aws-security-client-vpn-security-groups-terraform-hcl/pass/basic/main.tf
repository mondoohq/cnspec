# Compliant: client VPN endpoint restricts traffic with security groups.
resource "aws_ec2_client_vpn_endpoint" "pass_example" {
  description            = "example"
  server_certificate_arn = "arn:aws:acm:us-east-1:111122223333:certificate/abcd"
  client_cidr_block      = "10.0.0.0/16"
  vpc_id                 = "vpc-12345678"
  security_group_ids     = ["sg-0123456789abcdef0"]

  authentication_options {
    type                       = "certificate-authentication"
    root_certificate_chain_arn = "arn:aws:acm:us-east-1:111122223333:certificate/efgh"
  }

  connection_log_options {
    enabled = false
  }
}
