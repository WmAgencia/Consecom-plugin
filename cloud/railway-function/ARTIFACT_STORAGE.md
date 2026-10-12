# Consecom Cloud — arquivos privados (protótipo)

O serviço Railway `consecom-mcp-relay` aceita o envio e a recuperação de arquivos binários pelo mesmo domínio HTTPS do MCP, mas em endpoints separados da API MCP.

**Status:** deploy de teste concluído e startup self-test validou upload/download binário, hash, autenticação e exclusão. **Ainda não é multiusuário**, não deve ser compartilhado com clientes.

## Responsabilidades

- O **Claude Code** gera e manipula arquivos locais com suas próprias ferramentas e permissões.
- O **Consecom Proxy no Railway** transporta pedidos e resultados e, quando solicitado, recebe uma cópia de arquivo pronto.
- O **ChatGPT com widget MCP** recebe mensagens e publica respostas de texto por `consecom_chat_responder`. O uso da conversa ChatGPT como um backend de inferência compatível com Anthropic Messages não foi implementado nem validado.

## Transferência binária de arquivos

Todos os endpoints exigem `Authorization: Bearer <BRIDGE_INGEST_TOKEN>` nesta versão de testes (credencial privada da conta proprietária, **não deve ser compartilhada com clientes**).

| Método | Rota | Resposta |
| --- | --- | --- |
| POST | `/bridge/artifacts` | JSON com `id`, `file_name`, `mime`, `bytes`, `sha256`, `download_path` |
| GET | `/bridge/artifacts/{id}` | Os bytes originais e `Content-Disposition: attachment` |
| GET | `/bridge/artifacts/{id}/meta` | Metadados e SHA-256 |
| DELETE | `/bridge/artifacts/{id}` | Confirmação de exclusão |

Tipos aceitos: `application/pdf`, `image/png`, `image/jpeg`, `text/plain`, `application/zip`.

Limites de teste: até **8 MiB por arquivo** e **96 MiB em uso total**, considerando o volume Railway de 256 MiB compartilhado com SQLite. Um PDF precisa iniciar com a assinatura `%PDF-`, mas isso não garante que todo o documento seja válido; a validação profunda pertence ao cliente.

A URL de download **não é pública**. Sem Bearer, o servidor devolve HTTP 401. Não colocar tokens na URL nem registrar em logs.

### Exemplo para Windows PowerShell

```powershell
$base = "https://consecom-mcp-relay-production.up.railway.app"
$token = Read-Host "Token privado do Consecom de testes"
$headers = @{ Authorization = "Bearer $token"; "X-File-Name" = "banana.pdf" }

$artefato = Invoke-RestMethod -Uri "$base/bridge/artifacts" -Method Post -Headers $headers -ContentType "application/pdf" -InFile "C:\Projetos\banana.pdf"
$artefato | ConvertTo-Json

# Baixe em outra pasta para conferir o conteúdo:
Invoke-WebRequest -Uri "$base/bridge/artifacts/$($artefato.id)" -Headers @{Authorization = "Bearer $token"} -OutFile "C:\Projetos\banana-recuperada.pdf"
```

**Nunca inserir tokens reais no GitHub, nos arquivos públicos ou em mensagens de chat.**

## Qualidade e pendências

- ✅ Persistência SQLite em volume Railway, upload/download autenticados e resultado `self_test.status=passed` (20 verificações).
- ✅ Bloqueio HTTP 401 sem credencial, limite de tamanho e cota.
- ✅ SHA-256 para verificar transporte sem alteração dos bytes.
- ❌ Não há contas individuais com chaves próprias nem isolamento multi-tenant.
- ❌ Não existe envio automático do arquivo sandbox ChatGPT para Railway: o cliente deve chamar o upload explicitamente.
- ❌ Não existe integração da API Anthropic Messages `/v1/messages`, streaming ou `tool_use/tool_result` do Claude Code.
- ❌ Não há teste ponta a ponta real com Claude Code. O próximo teste deve conectar uma instância autorizada do IDE e verificar geração local, upload e recuperação.

## Observação sobre o ChatGPT

O widget consegue solicitar um turno quando carregado, mas a plataforma pode suspendê-lo. O servidor não desperta o ChatGPT quando a conversa está inativa e não pode garantir resposta contínua. O modelo comercial precisa de autorização apropriada antes de usar a sessão ChatGPT como serviço automatizado.
