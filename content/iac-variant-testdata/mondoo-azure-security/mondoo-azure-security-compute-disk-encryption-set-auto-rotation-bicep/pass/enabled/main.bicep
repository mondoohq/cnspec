resource des 'Microsoft.Compute/diskEncryptionSets@2023-04-02' = {
  name: 'des-cmk'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    encryptionType: 'EncryptionAtRestWithCustomerKey'
    activeKey: {
      keyUrl: 'https://contoso-kv.vault.azure.net/keys/disk-key/abc123'
    }
    rotationToLatestKeyVersionEnabled: true
  }
}
