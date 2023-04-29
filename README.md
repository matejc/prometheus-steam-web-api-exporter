# prometheus-steam-web-api-exporter
A prometheus exporter to create metrics from the Steam Web API

## Requirements

You need to obtain a Steam Web API Key. See https://partner.steamgames.com/doc/webapi_overview/auth

You also need the Steam IDs for the users you want to scrape. Usually it's just the number at the end of your profile URL, eg. https://steamcommunity.com/profiles/123/

## Usage

Just start the exporter with your API Key as an argument and the Steam IDs you want to get metrics for:

```
./prometheus-steam-web-api-exporter --steam-api-key 123456 --steam-ids "123,456"
```

You can use environment variables:

```
EXPORT STEAM_API_KEY="123456"
EXPORT STEAM_IDS="123,456"
./prometheus-steam-web-api-exporter
```

## Known Limitations

To obtain play time per steamid and game the API endpoint https://api.steampowered.com/IPlayerService/GetOwnedGames/v1 is used.
This endpoint doesn't provide information for shared games through family sharing. See issue #1.
