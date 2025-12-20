resource "aws_s3_bucket" "fail_example" {
  acl = "public-read-write" 
}


resource "aws_s3_bucket" "fail_example_2" {
  acl = "public-read-write"
}

resource "aws_s3_bucket_public_access_block" "fail_example_2" {
  bucket = aws_s3_bucket.example.id

  block_public_policy = false
}