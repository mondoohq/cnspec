resource "aws_iam_role_policy_attachment" "this" {
  role       = "read-only-role"
  policy_arn = "arn:aws:iam::aws:policy/ReadOnlyAccess"
}
