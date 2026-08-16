// TLS 1.0 is deprecated and disallowed by PCI DSS; accepting it lets a client
// negotiate down to it.
resource sqlMi 'Microsoft.Sql/managedInstances@2023-05-01-preview' = {
  name: 'prod-sqlmi'
  location: 'eastus'
  sku: {
    name: 'GP_Gen5'
    tier: 'GeneralPurpose'
  }
  properties: {
    administratorLogin: 'sqladmin'
    vCores: 4
    storageSizeInGB: 32
    minimalTlsVersion: '1.0'
  }
}
