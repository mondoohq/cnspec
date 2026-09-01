# Compliant: the organization centrally manages root credentials in member
# accounts and allows short-lived root sessions.
resource "aws_iam_organizations_features" "main" {
  enabled_features = ["RootCredentialsManagement", "RootSessions"]
}
