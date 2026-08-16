# Compliant: each user has a single active access key.
resource "aws_iam_access_key" "alice" {
  user = "alice"
}

resource "aws_iam_access_key" "bob" {
  user = "bob"
}
