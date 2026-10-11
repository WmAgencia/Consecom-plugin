# Consecom Plugin

Integração controlada entre o ChatGPT e um computador Windows por meio de um servidor MCP local e túnel privado.

**Versão atual do projeto:** 2.7.2

## Recursos

- Dashboard de métricas e discos, status de servidor e conexão.
- Laboratório de Agentes com registros de atividades reais.
- Leitura, gravação e organização de arquivos autorizados, com proteção para caminhos sensíveis e recuperação.
- Browser Control para Chrome/Edge: navegação, leitura de páginas, cliques permitidos, digitação, screenshots e múltiplas abas.
- Terminal de desenvolvimento restrito: Git e Node.js/npm; execução de scripts de projeto mediante aprovação local.
- Perfis e políticas de aprovação configuráveis no painel.
- Atualizador de um clique que utiliza exclusivamente o repositório oficial **WmAgencia/Consecom-plugin**, valida SHA-256 do instalador e reinicia o servidor.

## Mudanças na v2.7.2

A funcionalidade de **conversa/mensagens no painel foi removida**. Não há botão de conversa, painel de mensagens, formulário de envio de pedidos, rotas associadas nem ferramentas MCP para responder mensagens.

O arquivo local de histórico `task-inbox.json` de versões anteriores **não é apagado durante a atualização**, para evitar perda de dados; não é carregado pelo programa atual.

O painel detecta reinício para **recarregar a página automaticamente** após clicar em Atualizar. Na primeira migração da versão 2.7.1, recarregue a aba manualmente se a versão antiga não atualizar a página.

## Atualizações

Acesse `http://127.0.0.1:8765` no Windows e pressione **Atualizar** na parte inferior da barra lateral. O repositório é fixo: `WmAgencia/Consecom-plugin`. O próprio aplicativo consulta a última release, verifica integridade, inicia o instalador e tenta religar o túnel. Atualizações disparadas por comandos MCP mantêm as aprovações configuradas.

## Segurança

O painel escuta somente em `127.0.0.1:8765`. Arquivos do sistema, credenciais e áreas protegidas permanecem bloqueados. O plugin não fornece shell irrestrito; tarefas de execução de código exigem as permissões correspondentes. Não publique tokens, credenciais locais nem perfis do navegador neste repositório.

**Nota:** a assinatura ChatGPT não concede execução autônoma de tarefas em segundo plano diretamente pelo servidor local. O Consecom responde a ferramentas MCP chamadas por uma conversa já iniciada no ChatGPT.

Detalhes de instalação em [INSTALACAO_E_CONEXAO.md](INSTALACAO_E_CONEXAO.md).
