# Non-compliant: public hosted zone with no DNSSEC signing resource.
resource "aws_route53_zone" "example" {
  name = "example.com"
}
