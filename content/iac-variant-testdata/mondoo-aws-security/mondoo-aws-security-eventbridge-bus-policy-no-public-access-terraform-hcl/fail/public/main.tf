# Non-compliant: bus policy Allow statement grants access to any principal ("*") with no condition.
resource "aws_cloudwatch_event_bus_policy" "fail_example" {
  event_bus_name = "example-bus"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "PublicAccess"
        Effect    = "Allow"
        Principal = "*"
        Action    = "events:PutEvents"
        Resource  = "arn:aws:events:us-east-1:111122223333:event-bus/example-bus"
      }
    ]
  })
}
