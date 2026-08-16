# Compliant: image builder default internet access is disabled.
resource "aws_appstream_image_builder" "pass_example" {
  name          = "example-image-builder"
  image_name    = "AppStream-WinServer2019-example"
  instance_type = "stream.standard.medium"

  enable_default_internet_access = false
}
