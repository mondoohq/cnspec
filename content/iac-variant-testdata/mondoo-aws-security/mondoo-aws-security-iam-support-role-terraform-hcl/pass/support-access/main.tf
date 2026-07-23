resource "aws_iam_role_policy_attachment" "this" {
  role       = "support-role"
  policy_arn = "arn:aws:iam::aws:policy/AWSSupportAccess"
}
