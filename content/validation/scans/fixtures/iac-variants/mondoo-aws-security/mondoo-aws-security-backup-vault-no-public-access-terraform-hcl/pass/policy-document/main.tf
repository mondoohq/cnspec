data "aws_iam_policy_document" "vault" {
  statement {
    effect    = "Allow"
    actions   = ["backup:CopyIntoBackupVault"]
    resources = ["*"]

    principals {
      type        = "AWS"
      identifiers = ["arn:aws:iam::111122223333:root"]
    }
  }
}

resource "aws_backup_vault" "this" {
  name = "app-vault"
}

resource "aws_backup_vault_policy" "pass" {
  backup_vault_name = aws_backup_vault.this.name
  policy            = data.aws_iam_policy_document.vault.json
}
