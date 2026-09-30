app pomodoro em Go 👍

vai funcionar em macos quando eu comprar um mac

## Build no Windows

Execute `wails build` na pasta do projeto. O build fecha a versão aberta, remove
`build/bin/pomodoro.exe` e cria um novo executável com o mesmo nome. Se o app não
fechar normalmente, o build para antes de remover o arquivo.

As configurações, tarefas, histórico e despertadores ficam em
`%APPDATA%\pomodoro\data.json`. Esse arquivo não é apagado pelo build, então a
nova versão carrega os dados já salvos.
