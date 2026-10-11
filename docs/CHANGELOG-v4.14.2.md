# Consecom v4.14.2 — telemetria de latência por etapa (candidata)

Esta alteração é **posterior** à release oficial v4.14.1. A versão 4.14.1 permanece imutável.

## Instrumentação local

O Consecom passa a registrar, por ID de mensagem e sem duplicar o texto no diagnóstico:

1. `sent_at` — criação da mensagem na fila local.
2. `claimed_at` — reserva atômica por um único widget.
3. `followup_at` — instante registrado pelo widget imediatamente antes de chamar `window.openai.sendFollowUpMessage`.
4. `host_resolved_at` — resposta da promessa do host; **não significa resposta do ChatGPT concluída**.
5. `reply_at` — gravação real da resposta final via `consecom_chat_responder`.
6. `panel_displayed_at` — primeira renderização reconhecida pelo painel via endpoint local autenticado.

Campos ainda indisponíveis nas mensagens legadas permanecem vazios, nunca são fabricados. Eventos e timestamps persistem no journal local existente, sem usar uma segunda conexão MCP ou API da OpenAI.

O diagnóstico fica disponível na nova ferramenta `consecom_chat_diagnostico` do MCP existente, no endpoint local autenticado `GET /api/live/latency` e no botão **Latência** do chat.

## Leitura correta dos tempos

As durações são expressas em milissegundos: `queue_ms`, `claim_to_followup_ms`, `followup_to_host_ms`, `host_to_reply_ms`, `followup_to_reply_ms`, `reply_to_display_ms`, `total_to_reply_ms` e `total_to_display_ms`.

**Não somar indiscriminadamente etapas sobrepostas.** A diferença entre envio ao host e publicação da resposta inclui tanto a plataforma quanto o processamento do modelo: não há como separar suas parcelas com precisão a partir dessa integração. A maior fase observável é inferida somente de segmentos mutuamente exclusivos com timestamps válidos.

O programa calcula média, mediana, mínimo e máximo apenas para mensagens efetivamente respondidas e com timestamps disponíveis. Respostas antigas sem telemetria nova continuam visíveis, mas sem estimativas artificiais das etapas intermediárias.

## Testes

- Persistência após reinicialização e idempotência de acknowledgment do painel.
- Timestamps incompletos e rejeição de relógios inválidos.
- Endpoints autenticados e leitura apenas via GET no diagnóstico.
- Regressão dos fluxos originais de claim, host, publicação e interface.

## Limites de teste

GitHub Actions verifica compilação e comportamento simulado em Windows. Medições de latência reais no ChatGPT Chat exigem que o usuário mantenha o Consecom Live ativo e envie novas mensagens pelo painel após instalar a versão. Não prometer melhoria quantitativa antes disso.
