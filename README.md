# Millennium Go

[![Test Status](https://github.com/fabiomatavelli/millennium-go/actions/workflows/test.yml/badge.svg)](https://github.com/fabiomatavelli/millennium-go/actions/workflows/test.yml)

Esta biblioteca tem o intuito de facilitar a integração com o ERP Millennium utilizando Go.

## Instalação

Para começar a utilizar o Millennium com Go, instale o Go e rode o `go get`:

```bash
go get -u github.com/fabiomatavelli/millennium-go
```

Isso irá baixar a instalar a biblioteca e suas dependências.

## Exemplo

No exemplo abaixo, iremos listar todas as filiais cadastradas no Millennium

```go
package main

import (
	"github.com/fabiomatavelli/millennium-go"
)

type Filial struct {
  Filial int    `json:"filial"`
  Codigo string `json:"cod_filial"`
  Nome   string `json:"nome"`
  CNPJ   string `json:"cnpj"`
}

func main() {
  var filiais []Filial
  client := millennium.NewClient(context.Background(), "http://192.168.1.1:6017", 30)

  // Login utilizando a sessão do Millennium
  err := client.Login("usuario", "senha", millennium.Session)
  if err != nil {
    panic(err)
  }

  total, err := client.Get("millenium.filiais.lista", url.Values{}, &filiais)

  if err != nil {
    panic(err)
  }

  if total > 0 {
    for _, filial := range filiais {
      fmt.Printf("Filial: %s CNPJ: %s", filial.Nome, filial.CNPJ)
    }
  }
}
```

## License

[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Ffabiomatavelli%2Fmillennium-go.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2Ffabiomatavelli%2Fmillennium-go?ref=badge_large)
