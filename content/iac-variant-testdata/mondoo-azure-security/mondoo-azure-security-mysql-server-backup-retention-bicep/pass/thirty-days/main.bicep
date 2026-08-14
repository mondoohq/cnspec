resource mysql 'Microsoft.DBforMySQL/servers@2017-12-01' = {
  name: 'app-mysql'
  location: 'eastus'
  sku: {
    name: 'GP_Gen5_2'
    tier: 'GeneralPurpose'
  }
  properties: {
    version: '5.7'
    sslEnforcement: 'Enabled'
    storageProfile: {
      storageMB: 51200
      backupRetentionDays: 30
      geoRedundantBackup: 'Enabled'
    }
  }
}
