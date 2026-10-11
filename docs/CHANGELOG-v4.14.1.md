# Consecom v4.14.1 — Otimização e medição de latência no chat

## Medições antes da alteração

Registro real observado na versão 4.14.0:
- `evt-000013` ("Tudo bem?"): criação 19:55:14Z, resposta 19:57:18Z — **124s**.
- `evt-000014` (banana): criação 19:59:39Z, resposta 20:00:22Z — **43s**.
- `evt-000015` (modelo): criação 20:02:22Z, resposta 20:02:55Z — **33s**.

O atraso é variável e a maior parte ocorre depois da publicação da mensagem, durante a retomada/execução do turno no ChatGPT. A atualização pode reduzir o overhead controlado pelo Consecom, mas **não promete resposta instantânea ou latência estável da plataforma**.

## Otimizações implementadas

1. O widget agora chama `window.openai.sendFollowUpMessage` **antes** de enviar a telemetria de tentativa, evitando possível competição pelo mesmo canal MCP antes de enviar a solicitação.
2. O texto de instrução que acompanha a mensagem foi reduzido, mantendo obrigação explícita de chamar `consecom_chat_responder` para publicar a resposta final e mensagens de andamento para tarefas longas.
3. Falhas transitórias de inicialização do MCP ao reservar a mensagem geram **até 3 novas tentativas**, com 300/600/1200ms de espera. Respostas definitivas `claimed=false` nunca geram novas tentativas, preservando deduplicação.
4. Falhas temporárias do long polling do componente passam a ser reconsultadas em **400ms**, depois 800/1600/2500ms, em vez de esperar 2,5s após qualquer falha.
5. O painel consulta o histórico local a cada **900ms** em vez de 1.800ms, enquanto a página está visível.
6. Cada mensagem de resposta do assistente exibe tempo real decorrido entre envio e publicação, em segundos, para permitir medir ganho de desempenho com evidências.

## Como medir depois

1. Atualizar Consecom na Central de Controle para v4.14.1; atualizar/refresh plugin Consecom no ChatGPT.
2. Abrir um único Consecom Live no ChatGPT e mantê-lo ativo.
3. Perguntar pelo painel: "Responda somente com banana". Cronometrar da mensagem ao balão de retorno.
4. Repetir 5 vezes e comparar a mediana com v4.14.0; não inferir melhoria de uma medição isolada.

**Limites:** sem API paga, o ChatGPT depende do host/browser ativo. Não há controle via Consecom sobre a fila interna do serviço, a velocidade do modelo ou as políticas do cliente ChatGPT. As melhorias são locais e conservadoras.