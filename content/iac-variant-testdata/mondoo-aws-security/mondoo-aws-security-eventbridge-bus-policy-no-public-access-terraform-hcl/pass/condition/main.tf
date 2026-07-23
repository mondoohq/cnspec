# Compliant: wildcard principal is scoped by an org-id condition, so access is not public.
resource "aws_cloudwatch_event_bus_policy" "pass_example" {
  event_bus_name = "example-bus"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "AllowOrgAccounts"
        Effect    = "Allow"
        Principal = "*"
        Action    = "events:PutEvents"
        Resource  = "arn:aws:events:us-east-1:111122223333:event-bus/example-bus"
        Condition = {
          StringEquals = {
            "aws:PrincipalOrgID" = "o-abcd1234"
          }
        }
      }
    ]
  })
}
