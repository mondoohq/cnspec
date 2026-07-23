# Non-compliant: rejected patches may still be installed as dependencies.
resource "aws_ssm_patch_baseline" "allow" {
  name                    = "prod-baseline"
  operating_system        = "AMAZON_LINUX_2"
  rejected_patches        = ["kernel*"]
  rejected_patches_action = "ALLOW_AS_DEPENDENCY"
}
