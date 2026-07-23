# Non-compliant: resolver query log config exists but is never associated with a VPC.
resource "aws_route53_resolver_query_log_config" "fail_example" {
  name            = "example"
  destination_arn = "arn:aws:logs:us-east-1:111122223333:log-group:/aws/route53resolver:*"
}

# The check applies to VPC-scoped resolver query logging, so the terraform-hcl
# filter requires a VPC to be present in the configuration.
resource "aws_vpc" "example" {
  cidr_block = "10.0.0.0/16"
}
