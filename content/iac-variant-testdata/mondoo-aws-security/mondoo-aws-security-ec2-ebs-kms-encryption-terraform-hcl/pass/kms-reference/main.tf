resource "aws_kms_key" "ebs" {
  description             = "CMK for EBS volume encryption"
  deletion_window_in_days = 10
}

resource "aws_ebs_volume" "encrypted_cmk" {
  availability_zone = "us-east-1a"
  size              = 40
  encrypted         = true
  kms_key_id        = aws_kms_key.ebs.arn
}
