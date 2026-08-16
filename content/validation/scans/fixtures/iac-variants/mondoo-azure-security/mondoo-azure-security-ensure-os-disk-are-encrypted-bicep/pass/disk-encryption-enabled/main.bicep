resource dataDisk 'Microsoft.Compute/disks@2023-04-02' = {
  name: 'app-data-disk-01'
  location: 'eastus'
  sku: {
    name: 'Premium_LRS'
  }
  properties: {
    creationData: {
      createOption: 'Empty'
    }
    diskSizeGB: 128
    encryptionSettingsCollection: {
      enabled: true
      encryptionSettings: [
        {
          diskEncryptionKey: {
            secretUrl: 'https://contoso-kv.vault.azure.net/secrets/disk-key/0a1b2c'
            sourceVault: {
              id: '/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-vm/providers/Microsoft.KeyVault/vaults/contoso-kv'
            }
          }
        }
      ]
    }
  }
}
