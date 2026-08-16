resource sqlServer 'Microsoft.Sql/servers@2023-05-01-preview' = {
  name: 'sql-prod-eastus'
  location: 'eastus'
  properties: {
    administratorLogin: 'sqladmin'
    version: '12.0'
  }
}

resource threatPolicy 'Microsoft.Sql/servers/securityAlertPolicies@2023-05-01-preview' = {
  parent: sqlServer
  name: 'Default'
  properties: {
    state: 'Enabled'
    disabledAlerts: [
      'Sql_Injection'
      'Access_Anomaly'
    ]
    emailAddresses: [
      'secops@contoso.com'
    ]
    retentionDays: 30
  }
}
