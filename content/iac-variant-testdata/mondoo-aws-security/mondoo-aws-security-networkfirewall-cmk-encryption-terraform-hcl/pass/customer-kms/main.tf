resource "aws_networkfirewall_firewall" "perimeter" {
  name                = "perimeter"
  firewall_policy_arn = aws_networkfirewall_firewall_policy.perimeter.arn
  vpc_id              = aws_vpc.prod.id

  encryption_configuration {
    type   = "CUSTOMER_KMS"
    key_id = aws_kms_key.networkfirewall.arn
  }

  subnet_mapping {
    subnet_id = aws_subnet.firewall_a.id
  }
}
