package main

import "flag"

func processFlags() {
	addrDescription := "Endereço para acessar o serviço. Será utilizado nas requisições internas das páginas e para acesso às telas de operação. " +
		"Inclua o endereço todo, incluindo o esquema (http/https) e porta, se houver necessidade. Isto não muda a porta de execução da aplicação"
	addrPtr := flag.String("endereco", "", addrDescription)
	flag.Parse()
	flagsUsed.Location = *addrPtr
}
