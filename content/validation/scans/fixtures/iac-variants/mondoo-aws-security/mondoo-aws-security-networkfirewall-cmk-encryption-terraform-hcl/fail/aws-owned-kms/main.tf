# AWS_OWNED_KMS_KEY is the default; the firewall's rule and flow state are then
# encrypted with a key the account has no control over.
resource "aws_networkfirewall_firewall" "perimeter" {
  name                = "perimeter"
  firewall_policy_arn = aws_networkfirewall_firewall_policy.perimeter.arn
  vpc_id              = aws_vpc.prod.id

  encryption_configuration {
    type = "AWS_OWNED_KMS_KEY"
  }

  subnet_mapping {
    subnet_id = aws_subnet.firewall_a.id
  }
}
