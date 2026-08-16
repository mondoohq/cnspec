# Internet egress is on, so a subverted dependency can send whatever the function
# can read to an arbitrary destination.
resource "alicloud_fcv3_function" "processor" {
  function_name   = "processor"
  runtime         = "python3.10"
  handler         = "index.handler"
  memory_size     = 512
  internet_access = true

  code {
    oss_bucket_name = "example-code-bucket"
    oss_object_name = "processor.zip"
  }
}
