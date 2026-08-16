# Non-compliant: client VPN endpoint has no security groups attached.
resource "aws_ec2_client_vpn_endpoint" "fail_example" {
  description            = "example"
  server_certificate_arn = "arn:aws:acm:us-east-1:111122223333:certificate/abcd"
  client_cidr_block      = "10.0.0.0/16"

  authentication_options {
    type                       = "certificate-authentication"
    root_certificate_chain_arn = "arn:aws:acm:us-east-1:111122223333:certificate/efgh"
  }

  connection_log_options {
    enabled = false
  }
}
