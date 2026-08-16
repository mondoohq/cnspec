resource storageAccount 'Microsoft.Storage/storageAccounts@2023-01-01' = {
  name: 'stcontosodev02'
  location: 'eastus'
  sku: {
    name: 'Standard_LRS'
  }
  kind: 'StorageV2'
  properties: {
    minimumTlsVersion: 'TLS1_2'
    supportsHttpsTrafficOnly: true
  }
}

resource fileService 'Microsoft.Storage/storageAccounts/fileServices@2023-01-01' = {
  parent: storageAccount
  name: 'default'
  properties: {
    protocolSettings: {
      smb: {
        versions: 'SMB2.1;SMB3.0;SMB3.1.1'
        authenticationMethods: 'NTLMv2;Kerberos'
        channelEncryption: 'AES-256-GCM'
      }
    }
  }
}
