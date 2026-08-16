# The key name gives nothing away, but the value is an Alibaba access key ID. Scanning
# key names alone would miss this, which is why the values are scanned as well.
resource "alicloud_fcv3_function" "processor" {
  function_name   = "processor"
  runtime         = "python3.10"
  handler         = "index.handler"
  memory_size     = 512
  internet_access = false

  environment_variables = {
    UPSTREAM_ID = "LTAI5tExampleNotReal1234"
    LOG_LEVEL   = "info"
  }

  code {
    oss_bucket_name = "example-code-bucket"
    oss_object_name = "processor.zip"
  }
}
