# GoBot

![Go](https://github.com/z0rr0/gobot/workflows/Go/badge.svg)
![Version](https://img.shields.io/github/tag/z0rr0/gobot.svg)
![License](https://img.shields.io/github/license/z0rr0/gobot.svg)

[Vk Teams](https://biz.mail.ru/myteam/) messenger goBot. 
Common API [docs](https://myteam.mail.ru/botapi/).

## Build

```shell
make build
```

### Run

Config example file is [config.example.json](https://github.com/z0rr0/gobot/blob/main/config.example.json).

Local:

```shell
./gobot -config <CONFIG>
```

Docker:

```shell
docker run -d --name gobot --volume $PWD/data:/data/gobot --restart on-failure z0rr0/gobot:latest
```

### Commands

```
go - вернет участников чата в случайном порядке (алиас "/shuffle")
version - покажет текущую версию бота
link - добавит ссылку на звонок для чата (без параметров вернет текущую ссылку)
reset - удалит ссылку на звонок для чата
exclude - добавит пользователей из чата в список исключений (без параметров вернет список исключений)
include - удалит указанных пользователей из списка исключений (без параметров работает как "/go")
```

## License

This source code is governed by a GPLv3 license that can be found
in the [LICENSE](https://github.com/z0rr0/gobot/blob/main/LICENSE) file.
