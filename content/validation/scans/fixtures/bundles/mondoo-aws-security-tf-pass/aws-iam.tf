resource "aws_iam_role_policy" "test_policy" {
  name = "test_policy"
  role = aws_iam_role.test_role.id
  policy = data.aws_iam_policy_document.s3_policy.json
}

resource "aws_iam_role" "test_role" {
  name = "test_role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Sid    = ""
        Principal = {
          Service = "s3.amazonaws.com"
        }
      },
    ]
  })
}

data "aws_iam_policy_document" "s3_policy" {
  statement {
    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::${data.aws_caller_identity.current.account_id}:root"]
    }
    actions   = ["s3:GetObject"]
    resources = [aws_s3_bucket.example.arn]
  }
}

resource "aws_iam_policy" "policy" {
  name        = "test_policy"
  path        = "/"
  description = "My test policy"

  # Terraform's "jsonencode" function converts a
  # Terraform expression result to valid JSON syntax.
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = [
          "ec2:Describe*",
        ]
        Effect   = "Deny"
        Resource = "*"
      },
    ]
  })
}

resource "aws_iam_policy" "limit_access_keys" {
  name        = "LimitAccessKeys"
  description = "Ensure IAM users have at most one active access key"

  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Effect   = "Deny",
        Action   = "iam:CreateAccessKey",
        Resource = "*",
        Condition = {
          NumericGreaterThan = {
            "iam:AccessKeysCount": 1
          }
        }
      }
    ]
  })
}

resource "aws_iam_policy" "enforce_mfa" {
  name        = "EnforceMFA"
  description = "Require MFA for all IAM users with console access"

  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Effect   = "Deny",
        Action   = [
          "ec2:*",
          "s3:*",
          "iam:*",
          "lambda:*"
        ],
        Resource  = "*",
        Condition = {
          Bool = {
            "aws:MultiFactorAuthPresent": "false"
          }
        }
      }
    ]
  })
}
