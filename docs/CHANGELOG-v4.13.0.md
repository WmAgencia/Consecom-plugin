# Consecom Plugin v4.13.0 — Mensagens do painel para o ChatGPT Chat

## Objetivo

Enviar texto escrito pelo usuário no painel local Consecom (`http://127.0.0.1:8765`) diretamente à **conversa ChatGPT na qual o Consecom Live está aberto**, usando o **mesmo MCP seguro e conectado**.

Não instalar um segundo plugin. Não usar ChatGPT Work, API da OpenAI, automação do navegador nem serviços pagos.

## Como usar

1. Atualize o Consecom para v4.13.0 no botão `Atualizar` da Central de Controle.
2. No ChatGPT, atualize a lista das ferramentas da conexão Consecom existente. Se necessário, abra uma conversa nova com o mesmo plugin.
3. Peça **`Use Consecom, abra consecom_live_abrir`** (modo padrão `mensagens`). Mantenha o componente Consecom Live visível/ativo durante o teste.
4. No painel do Windows, abra **Central de Controle → Painel → Mensagem para o ChatGPT**, escreva uma frase e pressione **Enviar ao ChatGPT**.
5. A mensagem recebe um ID `evt-NNNNNN`, fica em `Aguardando ChatGPT`, passa a `Em envio` quando o widget reserva o evento, e a `Aceita pelo host` caso a tentativa seja aceita.
6. A resposta aparece na conversa do ChatGPT conectada àquele componente. **Host aceitou** não prova leitura ou resposta do modelo; confirme no próprio chat.

## Entrega e limitações

- O painel local usa a proteção CSRF `X-Local-Key`, validação de Origin e o servidor que escuta somente `127.0.0.1:8765`.
- Mensagens têm limite de **2000 bytes**, no máximo uma a cada **3 segundos**, UTF-8 válido, sem controles perigosos.
- A fila é salva em `live-lab-v412.json` e sobrevive ao reinício do processo.
- Quando o componente não está aberto ou o ChatGPT fica suspenso, a mensagem permanece na fila. **Não existe garantia de entrega em segundo plano com o navegador fechado.**
- Ao buscar a fila, o widget só pode **reservar cada mensagem uma vez** (claim atômico e persistente). Múltiplas janelas não conseguem reservar simultaneamente a mesma mensagem.
- Para evitar duplicação, mensagem `Em envio` só permite tentativa manual após 2 minutos e confirmação local no painel. Uma mensagem `Aceita pelo host` não é reenviada automaticamente.
- `consecom_live_abrir` agora inicia apenas a ponte de mensagens; use `consecom_live_abrir({modo:"teste"})` para reativar os nove erros fictícios do laboratório.
- O widget atende ao máximo de 50 tentativas por sessão; reabra-o para sessões adicionais.
- Nenhuma mensagem é executada como comando local automaticamente; permissões do Consecom para editar arquivos/Windows continuam independentes.
- A tela mostra os **12 envios recentes** e o ID/estado.
- O widget só envia à conversa na qual foi aberto. Se duas conversas tiverem componentes vivos, somente um widget conseguirá reivindicar o evento; o destino de uma fila comum não é determinado previamente. **Para testar, deixe apenas um componente Live aberto.**

## Status / confirmação

Através do mesmo MCP, `consecom_live_status`, `consecom_live_ler_evento` e `consecom_live_resultados` permitem verificar a fila e a resposta do host. A confirmação do modelo é observada na conversa, não no registro `host_resolved`.

## Risco residual

Se o host aceitar uma mensagem mas a resposta não chegar (ou um relatório de estado se perder durante falha de conexão), o painel pode ficar em `Em envio`. Existe botão de reenvio manual mediante confirmação; isso pode gerar duplicação no ChatGPT e não acontece automaticamente.
