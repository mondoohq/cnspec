# Compliant: the hosted zone has a matching DNSSEC signing resource.
resource "aws_route53_zone" "example" {
  name = "example.com"
}

resource "aws_route53_hosted_zone_dnssec" "example" {
  hosted_zone_id = aws_route53_zone.example.zone_id
}
