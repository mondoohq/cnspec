resource flowLog 'Microsoft.Network/networkWatchers/flowLogs@2023-09-01' = {
  name: 'NetworkWatcher_eastus/fl-nsg-data-prod'
  location: 'eastus'
  properties: {
    targetResourceId: resourceId('Microsoft.Network/networkSecurityGroups', 'nsg-data-prod')
    storageId: resourceId('Microsoft.Storage/storageAccounts', 'stflowlogsprod')
    retentionPolicy: {
      days: 90
      enabled: true
    }
  }
}
