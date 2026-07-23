# Compliant: resolver query log config exists and is associated with a VPC.
resource "aws_route53_resolver_query_log_config" "pass_example" {
  name            = "example"
  destination_arn = "arn:aws:logs:us-east-1:111122223333:log-group:/aws/route53resolver:*"
}

resource "aws_route53_resolver_query_log_config_association" "pass_example" {
  resolver_query_log_config_id = aws_route53_resolver_query_log_config.pass_example.id
  resource_id                  = "vpc-0123456789abcdef0"
}

# The check applies to VPC-scoped resolver query logging, so the terraform-hcl
# filter requires a VPC to be present in the configuration.
resource "aws_vpc" "example" {
  cidr_block = "10.0.0.0/16"
}
