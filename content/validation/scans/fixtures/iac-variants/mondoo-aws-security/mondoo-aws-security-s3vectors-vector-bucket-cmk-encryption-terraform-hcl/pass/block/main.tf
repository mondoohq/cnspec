# Vectors are encrypted under a customer managed key.
resource "aws_s3vectors_vector_bucket" "example" {
  vector_bucket_name = "rag-vectors"

  encryption_configuration {
    sse_type    = "aws:kms"
    kms_key_arn = "arn:aws:kms:us-east-1:111122223333:key/1234abcd-12ab-34cd-56ef-1234567890ab"
  }
}
