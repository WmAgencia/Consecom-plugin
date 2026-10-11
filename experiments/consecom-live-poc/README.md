# Consecom Live — Prova de conceito automática

Este diretório é um **experimento isolado**. Ele não altera a instalação atual
do Consecom, não acessa arquivos do Windows e não cria uma sessão oculta do
ChatGPT. O objetivo é verificar se o ChatGPT Chat aceita **ui/message** via
`window.openai.sendFollowUpMessage` disparado por um **evento SSE**, sem clique.

## O que está ligado

O widget vem com **“Solicitar continuação automaticamente” marcado**. Quando
chega um evento sintético novo, o componente tenta enviar o ID do evento ao
ChatGPT. O próprio host pode aceitar, solicitar consentimento ou rejeitar.
Nenhuma rejeição é contornada por click-simulation, cookies ou DOM scraping.

Restrições do protótipo:
- Somente `BUILD_FAILED` sintético. Nenhum dado pessoal ou comando do PC.
- O endpoint MCP expõe apenas ferramentas anônimas de **leitura**.
- `POST /demo/emit` requer um Bearer token local.
- O widget ignora eventos antigos para impedir reenvios após reconexão.
- No máximo 10 solicitações por sessão; intervalo mínimo 30 segundos.
- O resultado exibido após a chamada confirma apenas a entrega ao host,
  **não** confirma que o ChatGPT iniciou um turno ou chamou ferramentas.

## Testes locais

```powershell
$env:CONSECOM_LIVE_TOKEN = ([guid]::NewGuid().ToString("N") + [guid]::NewGuid().ToString("N"))
$env:CONSECOM_LIVE_LISTEN = "127.0.0.1:8793"
$env:CONSECOM_LIVE_PUBLIC_ORIGIN = "http://127.0.0.1:8793"
./Consecom-Live-POC.exe
```

Em outra janela:

```powershell
Invoke-RestMethod -Uri "http://127.0.0.1:8793/demo/emit" -Method Post -Headers @{Authorization = "Bearer $env:CONSECOM_LIVE_TOKEN"}
```

Use `go test ./experiments/consecom-live-poc` para executar os testes.

## Testar dentro do ChatGPT Chat

É preciso expor este processo por HTTPS (por exemplo, com um túnel
explicitamente configurado) e definir `CONSECOM_LIVE_PUBLIC_ORIGIN` como
a origem HTTPS pública. Conecte o endpoint `https://SEU_HOST/mcp`
como **MCP personalizado sem autenticação**, apenas para este experimento
sintético. Use o ChatGPT **Chat**, abra o widget pela ferramenta
`consecom_live_abrir`, e emita um novo evento via endpoint protegido.
Observe se o host inicia um turno e executa a ferramenta de leitura.

Se o host bloquear o disparo automático, **o experimento falhou nesta
superfície**. Não simule cliques para superar o bloqueio. A funcionalidade
de eventos oficiais MCP 2.0 no ChatGPT é documentada em superfícies Work,
não como ativação automática no Chat normal.

**Não exponha eventos reais da máquina por este endpoint anônimo.**
Qualquer versão real exigirá autenticação de usuário, escopos e auditoria.
Este protótipo não é um instalador v4.8 nem uma release oficial.
