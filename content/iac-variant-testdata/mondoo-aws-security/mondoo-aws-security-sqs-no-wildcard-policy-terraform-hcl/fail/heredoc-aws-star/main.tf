# Non-compliant: heredoc (raw-string) policy grants Principal.AWS = "*". This exercises
# the string-fallback path, which must catch "AWS": "*" as well as "Principal": "*".
resource "aws_sqs_queue_policy" "fail_example" {
  queue_url = "https://sqs.us-east-1.amazonaws.com/111122223333/example-queue"

  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PublicViaAws",
      "Effect": "Allow",
      "Principal": { "AWS": "*" },
      "Action": "sqs:SendMessage",
      "Resource": "arn:aws:sqs:us-east-1:111122223333:example-queue"
    }
  ]
}
POLICY
}
