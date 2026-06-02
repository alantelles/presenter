#!/bin/bash

# Função para processar e mover os arquivos
organizar_musica() {
    local arquivo="$1"
    local nome_base=$(basename "$arquivo")

    # Ignora arquivos que começam com '_' ou que não sejam arquivos comuns
    if [[ "$nome_base" == _* ]] || [[ ! -f "$arquivo" ]]; then
        return
    fi

    # Extrai a primeira letra e converte para maiúscula
    primeira_letra=${nome_base:0:1}
    primeira_letra=${primeira_letra^^}

    # Tratamento de acentuação (Normalização básica)
    # Transforma ÁÀÃÂ em A, ÉÊ em E, etc.
    primeira_letra=$(echo "$primeira_letra" | tr 'ÁÀÃÂÉÊÍÓÔÕÚÇ' 'AAAAEEIOOOUC')

    # Define o diretório de destino
    diretorio_destino="musicas/$primeira_letra"

    # Cria o diretório se não existir
    mkdir -p "$diretorio_destino"

    # Move o arquivo
    mv "$arquivo" "$diretorio_destino/"
    echo "Movido: $nome_base -> $diretorio_destino/"
}

ORIGEM=$(pwd)

cd media/songs
# 1. Processar arquivos .txt na raiz atual
for f in *.txt; do
    organizar_musica "$f" || echo "$f não pôde ser organizado com sucesso"
done

# 2. Processar arquivos .txt em musicas/_removidas/
if [ -d "musicas/_removidas" ]; then
    for f in musicas/_removidas/*.txt; do
        organizar_musica "$f" || echo "$f não pôde ser organizado com sucesso"
    done
fi

git add --all
git commit -m organizando
git push origin culto-de-hoje

cd $ORIGEM