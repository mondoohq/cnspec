# Non-compliant: access policy allows any principal via wildcard.
resource "aws_cloudwatch_log_destination_policy" "fail_example" {
  destination_name = "test_destination"
  access_policy = "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Principal\":\"*\",\"Action\":\"logs:PutSubscriptionFilter\",\"Resource\":\"arn:aws:logs:us-east-1:123456789012:destination:test_destination\"}]}"
}
