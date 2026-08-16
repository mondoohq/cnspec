# Non-compliant: image builder default internet access is enabled.
resource "aws_appstream_image_builder" "fail_example" {
  name          = "example-image-builder"
  image_name    = "AppStream-WinServer2019-example"
  instance_type = "stream.standard.medium"

  enable_default_internet_access = true
}
