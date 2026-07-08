# Non-compliant: counted repositories encrypted with AES256 instead of KMS.
resource "aws_ecr_repository" "counted" {
  count = 2
  name  = "counted-${count.index}"
  encryption_configuration {
    encryption_type = "AES256"
  }
}
