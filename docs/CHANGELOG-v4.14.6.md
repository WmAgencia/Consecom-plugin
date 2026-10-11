# Consecom Plugin v4.14.6 — Ativação de uma conversa por prompt

## Instalação do usuário
1. Instale o aplicativo Consecom Plugin no Windows normalmente e mantenha o servidor local/túnel iniciado.
2. Conecte o plugin MCP Consecom na conta ChatGPT pela interface oficial do ChatGPT.
3. No topo do painel Consecom, clique em **Copiar prompt de ativação**.
4. Abra uma conversa do ChatGPT com o plugin habilitado e cole o prompt.
5. O ChatGPT deve verificar o MCP e chamar `consecom_live_abrir(modo="mensagens")`, expondo o widget Live nessa conversa.
6. Deixe o widget aberto e a opção de recebimento automático marcada. Teste mensagens pelo painel Consecom.

## Mudanças técnicas
- Nova seção compacta de ativação antes do chat principal na Central de Controle.
- Prompt armazenado no HTML local, visualização em seção expansível, e botão de cópia pelo clipboard do navegador com fallback de seleção manual.
- Prompt referencia apenas ferramentas Consecom existentes: `atualizacao_status`, `consecom_chat_historico`, `consecom_live_abrir`, `consecom_chat_responder`.
- Mantidos painel, diagnóstico de latência, indicador de digitação, claim atômico, políticas e aprovação local.
- Testes de markup, presença das ferramentas, ausência de código do antigo benchmark e compilação JavaScript.

## Limite importante
O prompt não instala software no computador e **não habilita execução global/24h**. O componente depende da conversa ativa no ChatGPT e pode ser suspenso pelo host. Não foi adicionada API paga, ChatGPT Work, segunda conexão MCP ou automação do navegador.
