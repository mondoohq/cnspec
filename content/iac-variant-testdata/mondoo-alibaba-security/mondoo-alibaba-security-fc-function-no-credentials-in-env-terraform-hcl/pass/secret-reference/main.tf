# The environment names a secret; the function resolves the value at startup.
resource "alicloud_fcv3_function" "processor" {
  function_name   = "processor"
  runtime         = "python3.10"
  handler         = "index.handler"
  memory_size     = 512
  internet_access = false

  environment_variables = {
    DB_PASSWORD_SECRET = "example-db-password"
    LOG_LEVEL          = "info"
  }

  code {
    oss_bucket_name = "example-code-bucket"
    oss_object_name = "processor.zip"
  }
}
