# Compliant: access policy restricts the principal to a specific account.
resource "aws_cloudwatch_log_destination_policy" "pass_example" {
  destination_name = "test_destination"
  access_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { AWS = "123456789012" }
      Action    = "logs:PutSubscriptionFilter"
      Resource  = "arn:aws:logs:us-east-1:123456789012:destination:test_destination"
    }]
  })
}
