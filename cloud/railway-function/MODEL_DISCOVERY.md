# Consecom Cloud — Descoberta dinâmica de modelos

## O que foi implementado

O Consecom Gateway apresenta modelos em `GET /v1/models` no formato Anthropic e aceita `GET /v1/models/{id}`, com suporte a `limit`, `after_id`, `before_id` e filtro `lifecycle`.

Um mecanismo de registro dinâmico consulta a API `/v1/models` dos provedores **explicitamente configurados pelo administrador** no Railway, consolida modelos sem repetir identificadores e armazena o resultado em cache por 90 segundos. Até 8 provedores e 10 páginas de 100 modelos por provedor são consultados.

Os resultados são exibidos com campos `type=model`, `id`, `display_name`, `created_at`, `capabilities`, `max_input_tokens`, `max_tokens`, `lifecycle` e `deprecated_at`.

### Configuração dos provedores autorizados

As credenciais ficam exclusivamente nas variáveis privadas do serviço Railway, nunca embutidas no código ou expostas ao comprador.

Variável de catálogo `CONSECOM_UPSTREAM_PROVIDERS_JSON`, contendo lista JSON como esta:

```json
[
  {
    "name": "Provedor Autorizado",
    "base_url": "https://seu-provedor-compatível.example",
    "key_env": "CONSECOM_UPSTREAM_UM_KEY",
    "auth_scheme": "x-api-key"
  }
]
```

Variável separada `CONSECOM_UPSTREAM_UM_KEY` = **chave privada do provedor**.

O valor `base_url` é a URL base **sem** `/v1`. Só são aceitas URLs HTTPS com formato seguro. Esquemas `bearer` e `x-api-key` são suportados.

Ao receber `POST /v1/messages` com o ID de um modelo efetivamente descoberto, o gateway encaminha a requisição (incluindo `tool_use` / `tool_result` / streaming, se o upstream for compatível) para o provedor com a credencial guardada no Railway. A resposta é transmitida sem geração ou transformação do conteúdo do modelo.

**Atenção:** esta opção só funciona quando o administrador já possui acesso legítimo e configurou um provedor. Nenhum modelo pago é ativado por essa configuração sozinho.

### Aliases reconhecíveis no Claude Desktop

O Claude Desktop pode descartar IDs sem os nomes de família Claude no resultado de descoberta. Para diagnosticar a integração, além dos IDs históricos foram adicionados:

- `claude-consecom-connection-test`: resposta fixa de conectividade; **não é LLM**.
- `claude-consecom-cloud-bridge-test`: ponte de texto para widget ChatGPT ativo; **não é modelo Anthropic**.

Esses IDs são aliases de testes, não versões da Anthropic. **Não** adicionar IDs Claude oficiais que não estejam servidos por um backend autorizado.

### Status e diagnóstico

`GET /gateway/diagnostics` com `x-api-key` informa quais provedores estão configurados, status da descoberta, quantidade de modelos detectados e se existe inferência real efetivamente validada. Nunca exibe senhas.

O gateway atual, sem `CONSECOM_UPSTREAM_PROVIDERS_JSON`, continua atendendo apenas os **dois modos experimentais e seus dois aliases**, totalizando quatro IDs. Não há acesso a todos os modelos da Anthropic ou ChatGPT, pois um catálogo de nomes não concede autorização para inferência.

### Validar o catálogo pelo PowerShell

```powershell
$base = "https://consecom-mcp-relay-production.up.railway.app"
$key = Read-Host "Chave privada Consecom"
$headers = @{ "x-api-key" = $key }
Invoke-RestMethod "$base/v1/models?limit=1000" -Headers $headers
Invoke-RestMethod "$base/gateway/diagnostics" -Headers $headers
```

### O que falta para apresentar "verde" como inferência de verdade

- Claude Desktop precisa reconhecer ao menos um modelo com nome compatível, **e** poder executar sua inferência.
- Sem provedor real, `claude-consecom-connection-test` retorna texto fixo, não ferramentas, raciocínio ou Claude real.
- `claude-consecom-cloud-bridge-test` depende de widget ativo, fila única não-multi-tenant e não implementa chamadas de ferramentas estruturadas do Claude Code. Não usar como serviço comercial de inferência.
- Antes da comercialização, implementar usuários separados, segredo por cliente, auditoria e autorização para o modelo.

## Provas

Railway confirmou sucesso após implantação e testes internos `gateway-desktop-discovery-aliases`, `gateway-desktop-alias-message`, `gateway-models-list`, `gateway-model-details`, `gateway-dynamic-catalog-normalization`, `gateway-cloud-response-roundtrip`, OAuth e SQLite. **Ainda não houve verificação do indicador verde dentro da interface Claude Desktop do usuário após a nova publicação.**
