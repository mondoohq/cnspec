resource flowLog 'Microsoft.Network/networkWatchers/flowLogs@2023-09-01' = {
  name: 'NetworkWatcher_eastus/fl-nsg-web-prod'
  location: 'eastus'
  properties: {
    targetResourceId: resourceId('Microsoft.Network/networkSecurityGroups', 'nsg-web-prod')
    storageId: resourceId('Microsoft.Storage/storageAccounts', 'stflowlogsprod')
    enabled: false
    retentionPolicy: {
      days: 90
      enabled: true
    }
  }
}
