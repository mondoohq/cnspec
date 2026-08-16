# Compliant: private hosted zones are out of scope for DNSSEC (not internet-facing),
# so a zone with a vpc association is excluded from the check.
resource "aws_route53_zone" "private" {
  name = "internal.example.com"

  vpc {
    vpc_id = aws_vpc.example.id
  }
}

resource "aws_vpc" "example" {
  cidr_block = "10.0.0.0/16"
}
