# Non-compliant: public hosted zone has no query logging.
resource "aws_route53_zone" "fail_example" {
  name = "example.com"
}
