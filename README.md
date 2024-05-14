# main-frame backend
the interface that interacts with the raspberry pi HACKING modules

## Modules

### Wifi
:warning: __UNDERCONSTRUCTION__ :warning:\
Wifi module will be responsible of every thing related to wifi application 

#### Interfaces
Interfaces path will list device wirless interfaces, think of it as iwconfig.

#### ScanAP
ScanAP path will use a wireless interface to scan for access 
```console
curl -X POST \
     -d "{\"interfaceName\":\"wlan0\"}" \
     localhost
```

#### Deauth
:warning: __UNDERCONSTRUCTION__ :warning:\
Deauth path will deauthenticate users from an access point.\
Client mac can be specific mac address or keyword `all` *(case doesn't matter)* or left `empty` to run an automatic discovery and kick out clients.
>NOTE: client mac is string in the format of a mac address e.g. FF:FF:FF:FF:FF:FF

```console
curl -X POST \
     -d "{\"interfaceName\" : \"wlan0\",\
          \"apMac\" : \"AP_NAME\", \
          \"clientMac\" : \"all\", \
          \"safeClients\" : [\"safe:client:1\", \
                             \"safe:client:2\"]}" \
     localhost
```

#### Connect AP
ConnectAp path will connect you to an access point
```console
curl -X POST \
     -d "{\"apName\" : \"ssid\", \
          \"apPass\" : \"pass\"}" \
     localhost:port/api/v1/modules/wifi/connectAp/INTERFACE_NAME
```



>This module in an extraction of bettercap deauth module and an not claiming the code