// TDE is on but with the service-managed key, so the account cannot revoke
// access to the data by revoking the key.
resource tde 'Microsoft.Sql/managedInstances/encryptionProtector@2023-05-01-preview' = {
  name: 'current'
  properties: {
    serverKeyName: 'ServiceManaged'
    serverKeyType: 'ServiceManaged'
  }
}
