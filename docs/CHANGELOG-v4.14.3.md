# Consecom 4.14.3 — Bridge fast-path e diagnóstico de disponibilidade

## Meta de produto (não é SLA)

- **Até 10 segundos**, idealmente, entre `created_at` e `followup_at`.
- **Até 15 segundos** é o teto de referência, *com o ChatGPT Chat e widget Consecom Live efetivamente ativos*.
- O período `followup_at` → `reply_at` é registrado à parte: contém fila interna do host e geração do modelo, não inteiramente controláveis pelo Consecom.

## Hipótese técnica e correção

Na 4.14.2 o ciclo da UI iniciava `claimAndSend` sem aguardar sua conclusão e podia abrir um novo long-poll MCP de 8 segundos em paralelo. Se o host serializasse as chamadas, isso poderia atrasar a reivindicação, além da ida e volta de uma segunda ferramenta.

- Novo **fast-path atômico** no **mesmo MCP**: `consecom_live_aguardar_e_assumir`, com espera máxima de 6s, recebimento e `claimPanelMessage` persistente antes de retornar à UI.
- Resultado contém `claimed_ids`. O widget só chama `sendFollowUpMessage` para os IDs que essa chamada realmente reivindicou. Widgets simultâneos não duplicam envios.
- Em metadados de ferramenta antigos, o widget tenta fallback para `consecom_live_aguardar` + `consecom_live_assumir_mensagem`. Nesse caminho, aguarda a reivindicação antes de iniciar novo long-poll para evitar sobreposição.
- Não há novo MCP, API da OpenAI ou modelo auxiliar. `consecom_chat_responder` e a persistência continuam iguais.

## Estado de conexão no painel

- O servidor mantém somente o instante de **chamadas reais de polling** do widget (memória volátil), com limiar de 15 segundos.
- O histórico autenticado `/api/live/chat` inclui `connection.status` (`ready`, `passive` ou `disconnected`).
- `ready` significa que o servidor **observou recentemente** uma chamada automática fast-path. Não é uma garantia de que o host manterá execução em segundo plano.
- Se uma mensagem ficar mais de 15 segundos aguardando e nenhuma conexão automática recente for detectada, o painel orienta a reabrir Consecom Live.
- Nenhum heartbeat independente é enviado que possa sobrecarregar o MCP ou fingir que uma sessão ChatGPT suspensa está acordada.

## Testes e segurança

- Teste de 6 widgets concorrentes: exatamente um recebe o ID reivindicado.
- Teste de despertar do long-poll mediante criação de mensagem local.
- Replay e persistência do estado `claimed` após reinício.
- Compatibilidade com o protocolo antigo e observação correta de heartbeat passivo e expirado.
- JavaScript embutido com verificação de sintaxe; testes Go e build Windows no GitHub Actions.

### Medição de aceitação depois de instalar

Executar no painel 10 ou mais mensagens simples, com janela e Consecom Live ativos, esperando 3 segundos ou mais entre envios. Registrar `total` de criação→despacho, mediana, P90, pior e quantidade acima de 15s. Separar sessões com widget suspenso, sem esconder exceções. Nada é considerado aprovado com base apenas nos testes sintéticos ou CI.
