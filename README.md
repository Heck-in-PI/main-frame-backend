# main-frame backend
the interface that interacts with the raspberry pi HACKING modules

## Modules

### Wifi
Wifi module will be responsible of every thing related to wifi application (under construction)

#### Interfaces
Interfaces path will list device wirless interfaces, think of it as iwconfig.

#### ScanAP
ScanAP path will use a wireless interface to scan for access 
```console
curl -X POST 
     -d "{\"interfaceName\":""}"
     localhost
```

#### Deauth
:warning: __UNDERCONSTRUCTION__\
Deauth path will deauthenticate users from an access point.\
Client mac can be specific mac address or keyword `all` *case doesn't matter*

```console
curl -X POST 
     -d "{\"interfaceName\":"wlan0",
          \"apMac\":"AP_NAME",
          \"clientMac\":"all"}"
     localhost
```



>This module in an extraction of bettercap deauth module and an not claiming the code