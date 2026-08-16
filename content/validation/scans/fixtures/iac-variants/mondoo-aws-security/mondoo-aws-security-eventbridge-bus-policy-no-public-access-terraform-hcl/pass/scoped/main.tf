# Compliant: bus policy Allow statement scopes the principal to a specific account.
resource "aws_cloudwatch_event_bus_policy" "pass_example" {
  event_bus_name = "example-bus"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowScoped"
        Effect    = "Allow"
        Principal = { AWS = "arn:aws:iam::111122223333:root" }
        Action    = "events:PutEvents"
        Resource  = "arn:aws:events:us-east-1:111122223333:event-bus/example-bus"
      }
    ]
  })
}
