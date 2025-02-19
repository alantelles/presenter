package main

import "flag"

func processFlags() {
	addrDescription := "Endereço para acessar o serviço. Será utilizado nas requisições internas das páginas e para acesso às telas de operação. " +
		"Inclua o endereço todo, incluindo o esquema (http/https) e porta, se houver necessidade. Isto não muda a porta de execução da aplicação"
	addrPtr := flag.String("endereco", "", addrDescription)
	userDescription := "Usuário para autenticação básica. Deve ser informado junto com a senha"
	passDescription := "Senha para autenticação básica. Deve ser informada junto com o usuário"
	userPtr := flag.String("usuario", "admin", userDescription)
	passPtr := flag.String("senha", "admin", passDescription)
	flag.Parse()
	flagsUsed.Location = *addrPtr
	flagsUsed.Username = *userPtr
	flagsUsed.Password = *passPtr
}
