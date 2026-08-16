# Non-compliant: repository policy allows any AWS principal.
resource "aws_ecr_repository_policy" "fail_example" {
  repository = "fail-example"

  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PublicPull",
      "Effect": "Allow",
      "Principal": {
        "AWS": "*"
      },
      "Action": "ecr:GetDownloadUrlForLayer"
    }
  ]
}
POLICY
}
