# Two repositories; the second uses AES256 instead of KMS.
resource "aws_ecr_repository" "compliant" {
  name = "compliant"
  encryption_configuration {
    encryption_type = "KMS"
    kms_key         = "arn:aws:kms:us-east-1:111122223333:key/abcd"
  }
}

resource "aws_ecr_repository" "violating" {
  name = "violating"
  encryption_configuration {
    encryption_type = "AES256"
  }
}
