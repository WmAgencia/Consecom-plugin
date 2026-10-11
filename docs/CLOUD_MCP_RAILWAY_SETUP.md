# Consecom Cloud MCP — plano de implantação (Railway)

**Status (11/10/2026): bloqueado antes do provisionamento.** A conta Railway Primary, workspace `db4c8243-bac6-4bad-b723-8a930810f919` (mesmo workspace do `prospector-consecom`), retornou: `Free plan resource provision limit exceeded. Please upgrade to provision more resources!`

**O projeto `consecom-mcp-cloud` não foi criado. Nenhum serviço existente foi modificado.**

## Objetivo

Separar servidor MCP Cloud do Prospector. O servidor Cloud mantém fila de mensagens, interface MCP Apps e endpoint do proxy, reproduzindo apenas o **transporte bidirecional** do Consecom Live (MCP → widget ativo ChatGPT → resposta MCP). Não executa controle de desktop no host Railway.

## Código preparado

Um protótipo Go independente foi desenvolvido e testado fora do repositório, disponível como arquivo da conversa `Consecom-Cloud-MCP-prototipo.zip`, com:

- `cloud/mcp-server/main.go`: MCP JSON-RPC Streamable HTTP (initialize, tools/list, tools/call, resources), fila de transporte e endpoints /bridge;
- `cloud/mcp-server/widget.go`: componente MCP Apps com `sendFollowUpMessage` quando o host o disponibiliza;
- `cloud/mcp-server/main_test.go`: testes de autenticação, claim atômico, publicação de resposta e isolamento dos tokens;
- `cloud/mcp-server/Dockerfile`: compilação multi-stage Go 1.23;
- `cloud/mcp-server/README.md`: configuração, limites e riscos.

Comandos de teste local: `cd cloud/mcp-server && go test -race ./...`.

## Próximo passo no Railway

1. Liberar capacidade no workspace correto (upgrade ou remoção explícita e consciente de recursos ociosos, **sem** tocar automaticamente em Prospector).
2. Criar o projeto privado `consecom-mcp-cloud` na mesma workspace.
3. Publicar o serviço a partir de um repositório GitHub contendo o código acima, com `rootDirectory=cloud/mcp-server`, ou publicar o Dockerfile equivalente.
4. Configurar tokens fortes **diferentes** `MCP_ACCESS_TOKEN` e `BRIDGE_INGEST_TOKEN`, via Railway Variables (nunca versionar segredos).
5. Validar `GET /health`, `POST /mcp`, `POST /bridge/enqueue` e `GET /bridge/result/{id}` em ambiente controlado.
6. **Antes de qualquer cliente real:** implementar OAuth por conta, armazenamento durável (Postgres), isolamento multi-tenant, auditoria e cotas. A fila atual é em memória e não resiste a reinício.

## Distinção técnica obrigatória

O endereço Railway substitui o **túnel como ponto de entrada**, não os recursos Windows. Para abrir aplicativos no PC ainda será necessário um agente Windows e um canal autenticado de saída para o servidor, preservando todas as aprovações locais.

O MCP, por si só, não consegue despertar uma conversa ChatGPT suspensa. O componente Live é executado e controlado pelo host ChatGPT. O protótipo não oferece API de modelos Anthropic ou OpenAI e não substitui um serviço de inferência.
