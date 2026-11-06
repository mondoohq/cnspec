resource directLine 'Microsoft.BotService/botServices/channels@2022-09-15' = {
  name: 'example-bot/DirectLineChannel'
  location: 'global'
  properties: {
    channelName: 'DirectLineChannel'
    properties: {
      sites: [
        {
          siteName: 'default'
          isEnabled: true
          isV1Enabled: false
          isV3Enabled: true
          isSecureSiteEnabled: true
        }
      ]
    }
  }
}
