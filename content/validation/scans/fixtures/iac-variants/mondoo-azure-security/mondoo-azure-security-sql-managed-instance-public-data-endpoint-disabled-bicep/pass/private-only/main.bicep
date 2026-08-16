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
    publicDataEndpointEnabled: false
    minimalTlsVersion: '1.2'
  }
}
