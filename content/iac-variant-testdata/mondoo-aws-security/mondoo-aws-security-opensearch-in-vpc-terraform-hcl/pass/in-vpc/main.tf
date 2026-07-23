# Compliant: domain is deployed inside a VPC.
resource "aws_opensearch_domain" "pass_example" {
  domain_name = "example"

  vpc_options {
    subnet_ids = ["subnet-12345678"]
  }
}
