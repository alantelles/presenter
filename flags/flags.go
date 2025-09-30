package flags

import (
	"flag"
)

type FlagsSetup struct {
	Location           string
	TokenBibliaDigital string
	Username           string
	Password           string
}

var FlagsUsed FlagsSetup

func ProcessFlags() {
	locationPtr := GetLocationFlag()
	// TODO: remover
	// tokenBibliaDigitalTokenPtr := GetTokenBibliaDigitalTokenFlag()
	usernamePtr := GetUsernameFlag()
	passwordPtr := GetPasswordFlag()
	flag.Parse()
	FlagsUsed.Username = *usernamePtr
	FlagsUsed.Password = *passwordPtr
	FlagsUsed.Location = *locationPtr
	// TODO: remover
	// FlagsUsed.TokenBibliaDigital = *tokenBibliaDigitalTokenPtr
}

func GetLocationFlag() *string {
	description := "Endereço para acessar o serviço. Será utilizado nas requisições internas das páginas e para acesso às telas de operação. " +
		"Inclua o endereço todo, incluindo o esquema (http/https) e porta, se houver necessidade. Isto não muda a porta de execução da aplicação"
	return flag.String("endereco", "", description)
}

// TODO: remover
// func GetTokenBibliaDigitalTokenFlag() *string {
// 	description := "Token para acessar a API d'A Bíblia Digital (abibliadigital.com.br), " +
// 		"o provedor de textos bíblicos utilizado pelo aplicativo. " +
// 		"Se não for informado, o aplicativo não conseguirá acessar os textos bíblicos"
// 	return flag.String("tokenBibliaDigital", "", description)
// }

func GetUsernameFlag() *string {
	description := "Usuário para autenticação básica. Deve ser informado junto com a senha"
	return flag.String("usuario", "admin", description)
}

func GetPasswordFlag() *string {
	description := "Senha para autenticação básica. Deve ser informada junto com o usuário"
	return flag.String("senha", "admin", description)
}

func GetLocation() string {
	return FlagsUsed.Location
}

func GetTokenBibliaDigital() string {
	return FlagsUsed.TokenBibliaDigital
}

func GetUsername() string {
	return FlagsUsed.Username
}

func GetPassword() string {
	return FlagsUsed.Password
}
