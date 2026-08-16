resource securityContact 'Microsoft.Security/securityContacts@2020-01-01-preview' = {
  name: 'default'
  properties: {
    emails: 'secops@contoso.com;soc@contoso.com'
    phone: '+15551234567'
    alertNotifications: {
      state: 'On'
      minimalSeverity: 'High'
    }
    notificationsByRole: {
      state: 'On'
      roles: [
        'Owner'
      ]
    }
  }
}
