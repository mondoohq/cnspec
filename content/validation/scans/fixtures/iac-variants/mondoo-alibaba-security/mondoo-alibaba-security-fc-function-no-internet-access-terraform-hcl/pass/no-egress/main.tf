# The function talks only to services inside the VPC.
resource "alicloud_fcv3_function" "processor" {
  function_name   = "processor"
  runtime         = "python3.10"
  handler         = "index.handler"
  memory_size     = 512
  internet_access = false

  code {
    oss_bucket_name = "example-code-bucket"
    oss_object_name = "processor.zip"
  }
}
