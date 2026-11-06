resource bot 'Microsoft.BotService/botServices@2022-09-15' = {
  name: 'example-bot'
  location: 'global'
  kind: 'azurebot'
  sku: {
    name: 'S1'
  }
  properties: {
    displayName: 'example-bot'
    endpoint: 'https://example-bot.azurewebsites.net/api/messages'
    msaAppId: '00000000-0000-0000-0000-000000000000'
    msaAppType: 'MultiTenant'
  }
}
