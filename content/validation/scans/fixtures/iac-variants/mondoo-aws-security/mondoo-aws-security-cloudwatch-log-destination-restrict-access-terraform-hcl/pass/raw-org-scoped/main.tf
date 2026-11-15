# Compliant: wildcard principal scoped to one organization by aws:PrincipalOrgID,
# the only form of wildcard principal CloudWatch Logs accepts on a destination.
resource "aws_cloudwatch_log_destination_policy" "pass_example" {
  destination_name = "test_destination"
  access_policy    = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":\"*\",\"Action\":\"logs:PutSubscriptionFilter\",\"Resource\":\"arn:aws:logs:us-east-1:123456789012:destination:test_destination\",\"Condition\":{\"StringEquals\":{\"aws:PrincipalOrgID\":[\"o-a1b2c3d4e5\"]}}}]}"
}
