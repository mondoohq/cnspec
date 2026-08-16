# Non-compliant: Allow statement uses Principal.AWS = "*" (public) with no condition.
resource "aws_cloudwatch_event_bus_policy" "fail_example" {
  event_bus_name = "example-bus"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "PublicAccessViaAWS"
        Effect    = "Allow"
        Principal = { AWS = "*" }
        Action    = "events:PutEvents"
        Resource  = "arn:aws:events:us-east-1:111122223333:event-bus/example-bus"
      }
    ]
  })
}
