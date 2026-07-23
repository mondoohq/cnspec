resource "aws_iam_group_membership" "this" {
  name  = "team"
  group = "developers"
  users = []
}
