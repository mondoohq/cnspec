resource "aws_iam_user" "alice" {
  name = "alice"
}

resource "aws_iam_user" "bob" {
  name = "bob"
}

resource "aws_iam_group" "developers" {
  name = "developers"
}

resource "aws_iam_group_membership" "this" {
  name  = "team"
  group = aws_iam_group.developers.name
  users = [
    aws_iam_user.alice.name,
    aws_iam_user.bob.name,
  ]
}
