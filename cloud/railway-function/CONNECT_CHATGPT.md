# Como conectar Consecom MCP Cloud ao ChatGPT

## Endpoint

`https://consecom-mcp-relay-production.up.railway.app/mcp`

O projeto está na conta Railway **PROXY**, workspace de `wmagenciasuporte@gmail.com`, serviço `consecom-mcp-relay`, ambiente `production`.

## Criar plugin personalizado no ChatGPT

1. Abra Plugins → Adicionar servidor MCP personalizado.
2. Nome: **CONSECOM CLOUD**. Conexão: **URL do servidor**; insira o endpoint acima.
3. **Autenticação: OAuth**. Não escolha **Sem autenticação**, pois o servidor exige autorização.
4. Marque a confirmação de segurança e escolha **Criar como plugin**.
5. No fluxo de autorização aberto pelo ChatGPT, verifique que a página pertence a `consecom-mcp-relay-production.up.railway.app`.
6. No Railway, abra o serviço `consecom-mcp-relay` → **Variables** → copie o valor da variável **OAUTH_PAIRING_CODE** (não compartilhe essa variável, nem os tokens MCP/Bridge).
7. Cole o código na página **Conectar Consecom Cloud ao ChatGPT** e confirme.
8. No ChatGPT, instale/ative o plugin recém-criado e abra o componente chamando `consecom_live_abrir`.

## Compatibilidade OAuth implementada

- OAuth Protected Resource Metadata: `/.well-known/oauth-protected-resource` (+ `/mcp`).
- Authorization Server Metadata: `/.well-known/oauth-authorization-server`.
- Dynamic Client Registration: `POST /oauth/register`; aceita apenas callback HTTPS da origem `chatgpt.com` ou `chat.openai.com` para esta prova controlada.
- Autorização: `GET/POST /oauth/authorize` com código de pareamento, confirmação e PKCE S256.
- Token: `POST /oauth/token`, authorization_code e refresh_token com rotação.
- Tokens de acesso opacos guardados apenas em hash no SQLite persistente, expiração de 8 horas, refresh por 30 dias.
- Requisições MCP permanecem protegidas por Bearer; chamadas anônimas recebem HTTP 401 e challenge WWW-Authenticate.

## Provas realizadas em ambiente Railway

Em inicialização de produção, o serviço registrou `consecom-cloud-selftest status=passed` com:

- sqlite-integrity, mcp-unauthorized-401, mcp-initialize, mcp-tools-list
- bridge-enqueue, mcp-atomic-claim, mcp-response, bridge-result
- oauth-dcr, oauth-consent, oauth-mcp-access, oauth-refresh-rotation

Esse teste executa o protocolo no processo Railway, sem depender de um ChatGPT real. **Ainda é necessário testar a conexão na interface ChatGPT e a execução real do widget**.

## Restrições que não desapareceram

- O widget só consulta a fila quando está em execução na conversa ChatGPT. Não há mecanismo suportado para despertar a conversa suspensa a partir do Railway.
- O servidor remoto não se transforma em API de inferência de ChatGPT/Claude Code.
- O login por único código de pareamento é **somente para teste de proprietário**, não para revenda. Para múltiplos compradores é necessário login e autorização por usuário, revogação, auditoria e isolamento de dados.
- Os eventos são persistidos em volume Railway 256 MB. A fila pode crescer até 10.000 mensagens pendentes como limite de proteção, não há prazo de expiração por tempo para `queued`.
