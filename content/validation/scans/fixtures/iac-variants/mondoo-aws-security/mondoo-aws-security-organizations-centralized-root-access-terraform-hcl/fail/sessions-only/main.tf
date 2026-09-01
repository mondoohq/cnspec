# Non-compliant: root sessions are centralized, but member accounts keep
# their own long-lived root credentials.
resource "aws_iam_organizations_features" "main" {
  enabled_features = ["RootSessions"]
}
