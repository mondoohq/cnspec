# Compliant: rejected patches are blocked, not installed as dependencies.
resource "aws_ssm_patch_baseline" "block" {
  name                    = "prod-baseline"
  operating_system        = "AMAZON_LINUX_2"
  rejected_patches        = ["kernel*"]
  rejected_patches_action = "BLOCK"
}
