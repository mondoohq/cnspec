# Compliant: public hosted zone has query logging configured.
resource "aws_route53_zone" "pass_example" {
  name = "example.com"
}

resource "aws_route53_query_log" "pass_example" {
  zone_id                  = aws_route53_zone.pass_example.zone_id
  cloudwatch_log_group_arn = "arn:aws:logs:us-east-1:111122223333:log-group:/aws/route53/example:*"
}
