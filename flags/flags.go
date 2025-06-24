package flags

import (
	"flag"
)

type FlagsSetup struct {
	Location           string
	TokenBibliaDigital string
}

var FlagsUsed FlagsSetup

func ProcessFlags() {
	locationPtr := GetLocationFlag()
	tokenBibliaDigitalTokenPtr := GetTokenBibliaDigitalTokenFlag()
	flag.Parse()
	FlagsUsed.Location = *locationPtr
	FlagsUsed.TokenBibliaDigital = *tokenBibliaDigitalTokenPtr
}

func GetLocationFlag() *string {
	description := "Endereço para acessar o serviço. Será utilizado nas requisições internas das páginas e para acesso às telas de operação. " +
		"Inclua o endereço todo, incluindo o esquema (http/https) e porta, se houver necessidade. Isto não muda a porta de execução da aplicação"
	return flag.String("endereco", "", description)
}

func GetTokenBibliaDigitalTokenFlag() *string {
	description := "Token para acessar a API d'A Bíblia Digital (abibliadigital.com.br), " +
		"o provedor de textos bíblicos utilizado pelo aplicativo. " +
		"Se não for informado, o aplicativo não conseguirá acessar os textos bíblicos"
	return flag.String("tokenBibliaDigital", "", description)
}

func GetLocation() string {
	return FlagsUsed.Location
}

func GetTokenBibliaDigital() string {
	return FlagsUsed.TokenBibliaDigital
}
