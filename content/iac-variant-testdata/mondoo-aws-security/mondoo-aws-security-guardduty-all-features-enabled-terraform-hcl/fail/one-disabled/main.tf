resource "aws_guardduty_detector_feature" "s3" {
  detector_id = "abc"
  name        = "S3_DATA_EVENTS"
  status      = "ENABLED"
}
resource "aws_guardduty_detector_feature" "eks" {
  detector_id = "abc"
  name        = "EKS_AUDIT_LOGS"
  status      = "ENABLED"
}
resource "aws_guardduty_detector_feature" "malware" {
  detector_id = "abc"
  name        = "EBS_MALWARE_PROTECTION"
  status      = "ENABLED"
}
resource "aws_guardduty_detector_feature" "rds" {
  detector_id = "abc"
  name        = "RDS_LOGIN_EVENTS"
  status      = "ENABLED"
}
resource "aws_guardduty_detector_feature" "lambda" {
  detector_id = "abc"
  name        = "LAMBDA_NETWORK_LOGS"
  status      = "ENABLED"
}
resource "aws_guardduty_detector_feature" "runtime" {
  detector_id = "abc"
  name        = "RUNTIME_MONITORING"
  status      = "DISABLED"
}
