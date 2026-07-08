# Non-compliant: resolver query log config exists but is never associated with a VPC.
resource "aws_route53_resolver_query_log_config" "fail_example" {
  name            = "example"
  destination_arn = "arn:aws:logs:us-east-1:111122223333:log-group:/aws/route53resolver:*"
}
