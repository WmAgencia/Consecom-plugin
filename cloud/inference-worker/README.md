# Consecom Cloud — Motor de inferência local para Claude Code

**Status:** protótipo funcional de transporte e protocolo. O Railway valida registro do worker, descoberta do modelo, fila, resposta Anthropic com `tool_use` e serialização SSE. **Ainda falta testar o Ollama rodando no computador do usuário.**

## Arquitetura e limitação central

```
Claude Code/Claude Desktop
  │  (gateway URL + API Key de testes, x-api-key)
  ▼
Consecom Cloud (Railway, HTTPS /v1/messages)
  │  fila SQLite /worker/inference
  ▼
Worker local Node.js ⇄ Ollama (localhost:11434)
  │
  ▼
Modelo de linguagem local; ferramentas continuam no Claude Code
```

Este motor **NÃO usa a conversa do ChatGPT como backend de inferência**. O Consecom Live ChatGPT permanece uma ponte MCP experimental separada. Uma assinatura de ChatGPT não oferece uma API autorizada para servir respostas programáticas de modelos ao Claude Code.

## Requisitos na máquina Windows

- Node.js **20+**
- Ollama atualizado com suporte à Anthropic Messages API
- Modelo local instalado com suporte a ferramentas. Exemplo de teste, menor: `qwen3:4b` (qualidade/contexto podem ser insuficientes para projetos grandes); modelos maiores requerem mais RAM/VRAM.
- Acesso autorizado à conta Railway PROXY e à variável **CONSECOM_LOCAL_WORKER_TOKEN** de `consecom-mcp-relay`. Esta credencial é diferente da API Key do gateway e **não deve ser compartilhada com clientes**.

### Inicialização

Terminal PowerShell 1:

```powershell
ollama pull qwen3:4b
ollama list
```

O serviço Ollama deve estar acessível em `http://127.0.0.1:11434`.

Terminal PowerShell 2:

```powershell
# No repositório Consecom-plugin, branch feat/cloud-mcp-railway-starter
$env:CONSECOM_LOCAL_WORKER_TOKEN = Read-Host "Token secreto do worker Railway"
node .\cloud\inference-worker\worker.mjs
```

O worker consulta `/api/tags` para listar modelos instalados e os registra no Consecom Cloud. O ID público será gerado automaticamente no padrão `claude-local-<nome-do-modelo>`. Para `qwen3:4b`, normalmente `claude-local-qwen3-4b`. Veja o ID preciso no log do worker ou consulte `GET /v1/models` com a API Key do gateway.

O worker faz **somente inferência**, sem executar Bash, manipular projetos ou arquivos. Essa execução permanece sob as permissões do Claude Code.

## Configuração Claude Desktop

- Conexão: Gateway
- Credencial: chave de API estática do Consecom Gateway (`CONSECOM_GATEWAY_TEST_KEY` no Railway)
- URL: `https://consecom-mcp-relay-production.up.railway.app`
- Esquema: `x-api-key`
- Descoberta automática de modelos: habilitada
- Se o aplicativo não reconhecer o modelo descoberto, informe o ID que o worker anunciou na lista manual para teste

**Somente os campos do Claude Desktop são URL e API Key**. Um motor gratuito baseado em modelo local ainda exige um computador executando Ollama e o worker.

## Configuração no Claude Code CLI

```powershell
$env:ANTHROPIC_BASE_URL = "https://consecom-mcp-relay-production.up.railway.app"
$env:ANTHROPIC_API_KEY = Read-Host "API Key Consecom Gateway"
Remove-Item Env:\ANTHROPIC_AUTH_TOKEN -ErrorAction SilentlyContinue
$env:ANTHROPIC_MODEL = "claude-local-qwen3-4b"
claude --model $env:ANTHROPIC_MODEL
```

Claude Code preserva suas próprias ferramentas e decisões de execução; o modelo local deverá responder com `tool_use` quando apropriado. A documentação oficial Ollama marca `tool_choice`, metadados e contagem de tokens como não totalmente suportados, portanto **compatibilidade Claude Code não é garantida para todas as versões e recursos**.

## Verificação pelo PowerShell (sem executar comandos do IDE)

```powershell
$base="https://consecom-mcp-relay-production.up.railway.app"
$key=Read-Host "API Key do Gateway"
$h=@{"x-api-key"=$key;"anthropic-version"="2023-06-01"}
Invoke-RestMethod -Uri "$base/v1/models" -Headers $h | ConvertTo-Json -Depth 8

$payload=@{model="claude-local-qwen3-4b";max_tokens=128;stream=$false;messages=@(@{role="user";content="Responda em português: Consecom local conectado."})} | ConvertTo-Json -Depth 8
Invoke-RestMethod -Uri "$base/v1/messages" -Method POST -Headers $h -ContentType "application/json" -Body $payload
```

## APIs do worker (não expor token)

- `POST /worker/inference/register`: heartbeat e catálogo privado de modelos
- `POST /worker/inference/poll`: solicita tarefa pendente
- `POST /worker/inference/complete`: devolve mensagem Anthropic ou erro

Todas exigem `Authorization: Bearer <CONSECOM_LOCAL_WORKER_TOKEN>`. O worker **faz somente conexões de saída HTTPS**, de modo que nenhuma porta do Ollama precisa ser exposta publicamente.

## Segurança e limitações

- **Protótipo proprietário:** não há isolamento por usuário/tenant e apenas o worker `owner-local` é aceito. Não revender como multiusuário ainda.
- Limite 8 requisições simultaneamente pendentes na fila; resposta síncrona aguarda até ~55 segundos e stream até ~120 segundos.
- O Railway mantém temporariamente o conteúdo das solicitações em SQLite; evite enviar segredos de clientes durante testes.
- O worker local processa uma inferência por vez.
- Ollama local suporta parte do protocolo Anthropic; o worker usa `stream:false` e o Relay converte a resposta final em eventos SSE. Não há streaming incremental de tokens do modelo até o cliente.
- Testes internos Railway usaram **resposta simulada de um worker**, inclusive `tool_use`, **não um modelo Ollama real**.
- O Consecom Live ChatGPT não é parte do caminho de inferência local. O modo `claude-consecom-cloud-bridge-test` continua somente para mensagens simples via widget ativo e não suporta ferramentas nativas Claude Code.

Documentação oficial: https://docs.ollama.com/integrations/claude-code e https://docs.ollama.com/api/anthropic-compatibility
