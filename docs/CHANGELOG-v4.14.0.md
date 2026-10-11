# Consecom Plugin v4.14.0 — Chat bidirecional com respostas no Windows

## O que esta versão entrega

A Central de Controle (`http://127.0.0.1:8765/`) agora possui **Chat com ChatGPT**, com balões de conversa, histórico persistente, mensagens do usuário, respostas do assistente e atualizações de andamento durante trabalhos mais longos.

O mecanismo é **uma única conexão Consecom MCP**, sem ChatGPT Work, sem outra API OpenAI, nem um segundo plugin.

### Funcionamento

1. O usuário abre `consecom_live_abrir` em uma conversa ChatGPT Chat e mantém o widget ativo.
2. O usuário escreve uma mensagem na Central de Controle e pressiona Enter ou Enviar.
3. O Consecom salva a mensagem no journal protegido (`PANEL_MESSAGE`) e o widget MCP Apps a consome.
4. O widget chama `sendFollowUpMessage` para criar uma solicitação normal no ChatGPT Chat, usando a conversa na qual foi aberto.
5. O modelo lê e executa o pedido conforme os recursos e aprovações disponíveis no Consecom.
6. **Para retornar ao Windows**, o modelo precisa chamar explicitamente `consecom_chat_responder` usando `id`, `texto`, `tipo:"andamento"|"final"` e `chave` opcional de idempotência.
7. O painel consulta `GET /api/live/chat` a cada 1,8s e exibe as respostas no mesmo histórico, sem carregar conversas de outros serviços.

### Estados e limites

- Mensagens `queued` estão aguardando o widget; `claimed` e `host_resolved` significam tentativa de entrega/host, não resposta concluída; `answered` significa que o **modelo publicou uma resposta final no MCP**.
- O modelo pode postar mensagens `andamento` antes de uma resposta `final` para tarefas de maior duração.
- Um ID de evento só admite uma resposta final; `chave` opcional deduplica repetições de uma chamada.
- Entradas do usuário até 2.000 bytes e respostas até 6.000 bytes. Histórico persistente limitado a 50 eventos e 120 respostas recentes.
- A tela de chat não permite escrever respostas como se fossem do assistente; elas só entram por ferramentas MCP autenticadas. O dashboard só lê respostas e submete mensagens do usuário, com CSRF/Origin local.
- Ações no Windows continuam sujeitas às aprovações e escopos já estabelecidos.
- Se o ChatGPT não invocar `consecom_chat_responder`, a resposta não aparece no Windows, mesmo que apareça na conversa ChatGPT. **Este é o teste necessário para validar a versão.**
- Não há execução no ChatGPT quando o widget/conversa estiver fechado ou suspenso. O sistema não fornece sessão de ChatGPT headless.
- Com vários widgets/conversas abertos, a primeira janela que reivindicar uma mensagem a recebe; o sistema ainda não suporta roteamento por conversa individual. Para testar, use apenas uma conversa ChatGPT com o Consecom Live.
- As respostas no Consecom não são cópias passivas do ChatGPT: exigem uma chamada deliberada do modelo. Por isso podem ter diferenças de formatação se o modelo não reproduzir exatamente o texto.

### Como testar

Após instalar v4.14.0, atualize a lista de ferramentas do plugin Consecom no ChatGPT, **sem conectar outro plugin**. Abra `consecom_live_abrir`; no painel Consecom, pergunte por exemplo **“Quanto é 2 + 2?”**. O resultado esperado é uma bolha sua e, depois, uma bolha minha com **“4”**, publicada usando `consecom_chat_responder`.

Ao pedir uma implementação, espere ver uma bolha de andamento, execução pelas ferramentas permitidas e uma resposta final com resultados reais. O teste de resposta no painel só estará confirmado quando o modelo fizer a chamada à ferramenta e o painel exibir a resposta.
