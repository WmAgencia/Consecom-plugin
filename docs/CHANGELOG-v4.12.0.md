# Consecom Plugin 4.12.0 — Consecom Live LAB no mesmo MCP

**Uma conexão:** o Consecom já instalado no ChatGPT continua sendo o único MCP. O pacote agrega ferramentas `consecom_live_*` e o componente `ui://consecom/live/v1.html` à mesma autenticação Bearer e ao mesmo túnel. Nenhum outro endereço/serviço é necessário.

## Iniciar no ChatGPT Chat comum

Depois da atualização local:
1. Atualize/recarregue a lista de ferramentas do plugin Consecom conectado na interface do ChatGPT (essa descoberta é controlada pelo ChatGPT).
2. Na conversa, solicite: **Use o Consecom e abra `consecom_live_abrir`**.
3. O componente solicita `consecom_live_aguardar` via `window.openai.callTool` enquanto permanecer aberto, e tenta `sendFollowUpMessage` após cada evento novo.
4. O servidor gera no máximo 9 falhas de build **fictícias** a cada 25 segundos por sessão de 6 minutos.
5. Confira via MCP `consecom_live_resultados`: registra apenas `attempt`, `host_resolved`, `host_rejected`, `unavailable` ou `pending`, com ID do evento, sem conversas/logins/dados pessoais.

## O que esse experimento comprova

- O fluxo **servidor Consecom → evento → componente Live** pode funcionar dentro da conexão MCP já existente.
- Se a ponte do ChatGPT permitir chamadas originadas do componente, veremos o evento no UI e a tentativa de mensagem.
- **Somente uma nova mensagem gerada no ChatGPT, sem clique**, comprovará o objetivo de despertar a inteligência quando a conversa estiver ociosa.
- Um retorno `host_resolved` não comprova que um novo turno do modelo começou.
- A plataforma pode rejeitar `sendFollowUpMessage` sem gesto humano; isso será relatado, nunca contornado por automação de cliques.

## Segurança e limitações

- LAB habilitado somente quando o ChatGPT solicita a abertura, por uma janela limitada de tempo; no início não injeta eventos reais do Windows.
- Fila sintética guardada separadamente no arquivo `live-lab-v412.json`, preservando o arquivo `config.json` e as credenciais.
- O widget **não** usa Cloudflare público, WebSocket exterior, Chrome DevTools, cookies, API OpenAI nem Work. A comunicação acontece somente via `window.openai.callTool`.
- Novos recursos/descrições dependem da atualização ou redescoberta das ferramentas pelo ChatGPT. O modelo precisa invocar `consecom_live_abrir` para que o componente exista.
- Simulação não executa comandos no PC. Ações reais deverão usar ferramentas Consecom existentes, com as aprovações atuais, em implementação futura e revisada separadamente.
