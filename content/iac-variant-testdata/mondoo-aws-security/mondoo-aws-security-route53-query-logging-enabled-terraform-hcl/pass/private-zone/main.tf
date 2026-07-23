# Compliant: private hosted zone (has a vpc block) is out of scope for public
# query logging, so absence of a query_log resource is acceptable.
resource "aws_route53_zone" "private_example" {
  name = "internal.example.com"

  vpc {
    vpc_id = "vpc-0123456789abcdef0"
  }
}
