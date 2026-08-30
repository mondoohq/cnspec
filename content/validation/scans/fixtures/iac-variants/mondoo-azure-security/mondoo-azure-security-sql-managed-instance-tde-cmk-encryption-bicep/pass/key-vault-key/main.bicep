resource tde 'Microsoft.Sql/managedInstances/encryptionProtector@2023-05-01-preview' = {
  name: 'sqlmi-prod/current'
  properties: {
    serverKeyName: 'example-kv_sqlmi-cmk_0123456789abcdef'
    serverKeyType: 'AzureKeyVault'
    autoRotationEnabled: true
  }
}
