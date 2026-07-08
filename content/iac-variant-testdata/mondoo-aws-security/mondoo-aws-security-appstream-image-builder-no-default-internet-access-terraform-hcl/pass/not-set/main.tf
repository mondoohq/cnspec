# Compliant: default internet access is not set, so it defaults to disabled.
resource "aws_appstream_image_builder" "pass_example" {
  name          = "example-image-builder"
  image_name    = "AppStream-WinServer2019-example"
  instance_type = "stream.standard.medium"
}
