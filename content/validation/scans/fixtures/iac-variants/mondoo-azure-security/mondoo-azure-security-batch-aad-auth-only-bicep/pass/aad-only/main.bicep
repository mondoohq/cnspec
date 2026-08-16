resource batchAccount 'Microsoft.Batch/batchAccounts@2024-02-01' = {
  name: 'mybatchaccount'
  location: 'eastus'
  properties: {
    allowedAuthenticationModes: [
      'AAD'
    ]
    publicNetworkAccess: 'Disabled'
  }
}
