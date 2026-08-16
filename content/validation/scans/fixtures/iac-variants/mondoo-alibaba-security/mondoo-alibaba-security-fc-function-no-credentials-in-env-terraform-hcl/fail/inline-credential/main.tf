# The database password sits in the function configuration, where anyone who can
# describe the function can read it, and where it also lands in state and CI logs.
resource "alicloud_fcv3_function" "processor" {
  function_name   = "processor"
  runtime         = "python3.10"
  handler         = "index.handler"
  memory_size     = 512
  internet_access = false

  environment_variables = {
    DB_PASSWORD = "hunter2-not-a-real-password"
    LOG_LEVEL   = "info"
  }

  code {
    oss_bucket_name = "example-code-bucket"
    oss_object_name = "processor.zip"
  }
}
