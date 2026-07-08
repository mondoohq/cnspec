resource securityContact 'Microsoft.Security/securityContacts@2023-12-01-preview' = {
  name: 'default'
  properties: {
    emails: 'secops@contoso.com'
    phone: '+1-555-0100'
    isEnabled: true
    notificationsByRole: {
      state: 'On'
      roles: [
        'Owner'
      ]
    }
    alertNotifications: {
      state: 'On'
      minimalSeverity: 'High'
    }
  }
}
