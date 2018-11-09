# Millennium Go

[![Build Status](https://travis-ci.org/fabiomatavelli/millennium-go.svg?branch=master)](https://travis-ci.org/fabiomatavelli/millennium-go)
[![Maintainability](https://api.codeclimate.com/v1/badges/85c1d065ae5a2a15aff2/maintainability)](https://codeclimate.com/github/fabiomatavelli/millennium-go/maintainability)
[![Test Coverage](https://api.codeclimate.com/v1/badges/85c1d065ae5a2a15aff2/test_coverage)](https://codeclimate.com/github/fabiomatavelli/millennium-go/test_coverage)
[![Go Report Card](https://goreportcard.com/badge/github.com/fabiomatavelli/millennium-go)](https://goreportcard.com/report/github.com/fabiomatavelli/millennium-go)

Esta biblioteca tem o intuito de facilitar a integração com o ERP Millennium utilizando Go.

## Instalação

Para começar a utilizar o Millennium com Go, instale o Go e rode o `go get`:

```bash
go get -u github.com/fabiomatavelli/millennium-go
```

Isso irá baixar a instalar a biblioteca.

## Exemplo

No exemplo abaixo, iremos listar todas as filiais cadastradas no Millennium

```go
package main

import (
	millennium "github.com/fabiomatavelli/millennium-go"
)

type Filial struct {
  Filial int    `json:"filial"`
  Codigo string `json:"cod_filial"`
  Nome   string `json:"nome"`
  CNPJ   string `json:"cnpj"`
}

func main() {
  var filiais []Filial
	client := millennium.Client("192.168.1.1:6017", 30)

  total, err := client.Get("millenium.filiais.lista", url.Values{}, &filiais)
  
  if err != nil {
    fmt.Print(err)
  }

  if total > 0 {
    for _, filial := range filiais {
      fmt.Printf("Filial: %s CNPJ: %s", filial.Nome, filial.CNPJ)
    }
  }
}
```