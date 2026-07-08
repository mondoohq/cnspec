resource securityContact 'Microsoft.Security/securityContacts@2020-01-01-preview' = {
  name: 'default'
  properties: {
    phone: '+15551234567'
    alertNotifications: {
      state: 'On'
      minimalSeverity: 'High'
    }
  }
}
