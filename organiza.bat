@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

set "ORIGEM=%cd%"

:: Navega até a pasta das músicas
cd "media\songs"

:: 1. Processar arquivos .txt na raiz atual
for %%f in (*.txt) do (
    :: Passamos o caminho absoluto (%~f1) para não ter erro
    call :organizar_musica "%%~ff" "raiz"
)

:: 2. Processar arquivos .txt em musicas\_removidas\
if exist "musicas\_removidas" (
    cd "musicas\_removidas"
    for %%f in (*.txt) do (
        call :organizar_musica "%%~ff" "removidos"
    )
    :: Volta para a pasta songs para rodar o Git corretamente
    cd "..\.."
)

cd "%ORIGEM%"
exit /b

:: ==========================================
:: FUNÇÃO PARA PROCESSAR E MOVER OS ARQUIVOS
:: ==========================================
:organizar_musica
set "arquivo=%~1"
set "nome_base=%~nx1"
set "origem_loop=%~2"

if not exist "%arquivo%" exit /b

set "primeira_letra=%nome_base:~0,1%"
if "%primeira_letra%"=="_" exit /b

:: Converte para Maiúscula
for %%A in (A B C D E F G H I J K L M N O P Q R S T U V W X Y Z) do (
    if /i "%primeira_letra%"=="%%A" set "primeira_letra=%%A"
)

:: Tratamento de acentuação
if /i "%primeira_letra%"=="Á" set "primeira_letra=A"
if /i "%primeira_letra%"=="À" set "primeira_letra=A"
if /i "%primeira_letra%"=="Ã" set "primeira_letra=A"
if /i "%primeira_letra%"=="Â" set "primeira_letra=A"
if /i "%primeira_letra%"=="É" set "primeira_letra=E"
if /i "%primeira_letra%"=="Ê" set "primeira_letra=E"
if /i "%primeira_letra%"=="Í" set "primeira_letra=I"
if /i "%primeira_letra%"=="Ó" set "primeira_letra=O"
if /i "%primeira_letra%"=="Ô" set "primeira_letra=O"
if /i "%primeira_letra%"=="Õ" set "primeira_letra=O"
if /i "%primeira_letra%"=="Ú" set "primeira_letra=U"
if /i "%primeira_letra%"=="Ç" set "primeira_letra=C"

:: Se veio da pasta de removidos, precisamos garantir que o destino 
:: seja a pasta 'musicas' que está na raiz de 'songs', e não uma subpasta.
if "%origem_loop%"=="removidos" (
    set "diretorio_destino=..\!primeira_letra!"
) else (
    set "diretorio_destino=musicas\!primeira_letra!"
)

if not exist "!diretorio_destino!" mkdir "!diretorio_destino!"

move /y "%arquivo%" "!diretorio_destino!\" >nul
if %errorlevel% equ 0 (
    echo Movido: %nome_base% -^> musicas/!primeira_letra!/
) else (
    echo %nome_base% não pôde ser organizado com sucesso
)

exit /b