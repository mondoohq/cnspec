# Non-compliant: rejected_patches_action not set (defaults to ALLOW_AS_DEPENDENCY).
resource "aws_ssm_patch_baseline" "absent" {
  name             = "prod-baseline"
  operating_system = "AMAZON_LINUX_2"
  rejected_patches = ["kernel*"]
}
