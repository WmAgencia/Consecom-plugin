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
Start-Process -FilePath "./Consecom-Live-POC.exe"
Start-Sleep -Seconds 2
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


## Experimento de retorno durante uma chamada MCP ativa

A fila reversa armazena eventos **sintéticos** em disco com sequência monotônica.
O servidor escreve antes de entregar SSE aos espectadores. O endpoint
`POST /demo/emit` mantém autenticação Bearer; o MCP deste laboratório é
público e só lê dados sintéticos, **nunca** dados do Windows.

Há duas ferramentas adicionais:

- `consecom_live_estado_fila`: sequência, número de eventos e retenção.
- `consecom_live_aguardar_evento`: recebe `after_sequence` e `wait_ms`
  (0–12000). Mantém uma chamada **já iniciada pelo modelo** aguardando.
  Retorna `status=event` em evento novo; `timeout` quando nada chegou;
  `gap=true` se a sequência requerida já saiu do histórico retido.

No processo normal, o diário fica em um subdiretório dedicado
`ConsecomLivePOC/events.json` da pasta de configuração do usuário. É
possível usar `CONSECOM_LIVE_STATE_FILE` para apontar para arquivo
isolado de teste, com caminho absoluto.

### Teste local (não usa API da OpenAI)

1. Inicie o POC a partir do PowerShell conforme a seção anterior.
2. Faça uma requisição MCP `tools/call` com
   `consecom_live_aguardar_evento`, `after_sequence=0` e
   `wait_ms=12000`.
3. Enquanto ela aguarda, envie `POST /demo/emit` usando o token do laboratório.
4. A chamada MCP deve retornar `status=event` com o mesmo identificador que
   a transmissão SSE exibiu. Meça a diferença entre emissão e retorno.
5. Espere 12 segundos sem novos eventos e confirme `status=timeout`.
6. Reinicie o processo e valide que a sequência continua.

Testes automatizados Windows/Linux:
`go test -v ./experiments/consecom-live-poc`.
Linux usa também `-race`.

### Teste efetivo no ChatGPT Chat

Para provar a retomada **dentro de ChatGPT**, o POC precisa estar acessível
em HTTPS e conectado como **app MCP de testes**, separadamente do Consecom
de produção. Com o ChatGPT Chat aberto, inicie uma chamada
`consecom_live_aguardar_evento` e emita um evento simulado. A ferramenta
pode retornar a tempo de o modelo analisar seu resultado **no turno atual**.
Isso não demonstra que o servidor consegue criar outro turno.

Abra o painel `consecom_live_abrir` e emita outro evento; o widget tenta
`sendFollowUpMessage` automaticamente. Observe se o host bloqueia com
erro de gesto humano. **A aceitação sem clique NÃO está comprovada** por
compilação, teste HTTP, SSE nem pelo sucesso de envio ao host. O resultado
definitivo depende da execução da interface MCP no ChatGPT Chat.

Não há ChatGPT Work, automação de cookies, Playwright em chat.openai.com,
modelo local disfarçado de ChatGPT ou chamada paga à API da OpenAI.
A instalação atual do Consecom permanece inalterada.
