# Compliant intent: a feature flag toggles CMK vs default encryption; the
# active (default) value is KMS.
variable "use_cmk" {
  type    = bool
  default = true
}

resource "aws_ecr_repository" "ternary" {
  name = "ternary"
  encryption_configuration {
    encryption_type = var.use_cmk ? "KMS" : "AES256"
    kms_key         = "arn:aws:kms:us-east-1:111122223333:key/abcd"
  }
}
