# Consecom Cloud: gateway integrado ao relay (ensaio privado)

## O que foi conectado

`POST /v1/messages` reconhece `model: "consecom-cloud-bridge-test"` e envia uma mensagem de texto à fila persistente do MCP Cloud. O widget ativo do ChatGPT pode reservar e publicar a resposta usando `consecom_chat_responder`. O gateway acompanha o ID da mensagem e devolve a resposta recebida, no formato básico de `Messages`.

O sistema mantém o modelo separado `consecom-connection-test`, que responde texto fixo apenas para diagnóstico.

### Endpoints
- URL base: `https://consecom-mcp-relay-production.up.railway.app`
- Proxy: `POST /v1/messages`
- MCP: `POST /mcp`
- Fila: `POST /bridge/enqueue`; `GET /bridge/result/{id}` (com credencial própria do relay).
- Autenticação gateway: `Authorization: Bearer <CONSECOM_GATEWAY_TEST_KEY>`, definida no Railway.

### Exemplo PowerShell

```powershell
$env:ANTHROPIC_BASE_URL = "https://consecom-mcp-relay-production.up.railway.app"
$env:ANTHROPIC_AUTH_TOKEN = Read-Host "Insira a chave de testes do gateway"
$body = @{
    model = "consecom-cloud-bridge-test"
    max_tokens = 200
    messages = @(@{ role = "user"; content = "Este é um teste do gateway. Responda CONSECOM BRIDGE OK." })
} | ConvertTo-Json -Depth 8

Invoke-RestMethod -Uri "$env:ANTHROPIC_BASE_URL/v1/messages" -Method POST `
  -Headers @{ Authorization = "Bearer $env:ANTHROPIC_AUTH_TOKEN"; "anthropic-version" = "2023-06-01" } `
  -ContentType "application/json" -Body $body
```

## Limites deste protótipo

- Limite do texto de entrada: 1200 caracteres; até 12 mensagens na requisição, mas só o texto do último `user` é encaminhado.
- O modelo `consecom-cloud-bridge-test` **não é um modelo LLM**. O gateway aguarda uma resposta da conversa ChatGPT ativa por até 42 segundos; após isso retorna erro 504 e preserva a mensagem na fila.
- SSE experimental envia ping enquanto aguarda. Não há semântica completa do protocolo Claude Code nem processamento de chamadas `tool_use/tool_result`; solicitações com ferramentas são rejeitadas com 501.
- O ChatGPT pode suspender o widget. O servidor MCP não pode despertá-lo independentemente.
- **A conexão automática da sessão ChatGPT para uso como API comercial de inferência não é um recurso suportado.** Não distribuir como serviço de modelos.
- Autenticação do gateway usa token proprietário de teste; não existe isolamento de clientes nesta versão.

## Teste realizado no Railway

No startup, uma chamada autenticada ao gateway foi encaminhada à fila. Os próprios handlers do MCP reservaram o evento, publicaram uma resposta simulada (`BRIDGE_INTERNAL_ROUNDTRIP_OK`) e o gateway devolveu essa resposta. Todos os passos constaram como `self_test.status=passed`, inclusive `gateway-cloud-enqueue`, `gateway-cloud-mcp-claim` e `gateway-cloud-response-roundtrip`.

Esse teste demonstra transporte **interno**. Não prova que o Claude Code consegue usar as ferramentas de multiagentes nem que um widget ChatGPT suspenso responderá sozinho.
