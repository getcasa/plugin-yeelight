# Yeelight
This plugin is a part of [Casa](https://github.com/getcasa), it's used to interact with yeelight ecosystem.

## Downloads
Use the integrated store in casa or [github releases](https://github.com/getcasa/plugin-yeelight/releases).

## Build
```
sudo env CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc go build -buildmode=plugin -o yeelight.so *.go
```

## Install
1. Extract `yeelight.zip`
2. Move `yeelight` folder to casa `plugins` folder
3. Restart casa
