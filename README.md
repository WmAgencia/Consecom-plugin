# Consecom Plugin — versão 2.6

Controle do Windows pelo ChatGPT através de um MCP próprio e túnel oficial da OpenAI.

## O que mudou

- Identidade **Consecom Plugin** no painel e novos atalhos. Mantemos internamente `ConsecomMCP`, `MCP-PC.exe` e o protocolo MCP por compatibilidade com a instalação existente.
- **Atualizar agora** no painel: usa GitHub Releases do repositório confiável configurado, verifica versão e SHA-256 e instala após aprovação local ou pré-autorização explícita.
- **Terminal de desenvolvimento controlado** para projetos autorizados: Git status/branch/diff stat/log, Node/npm version. npm test/build/lint sempre exigem confirmação **no PC**, pois scripts podem executar código arbitrário.
- **Navegador profissional, fase 1:** listar/criar/ativar múltiplas abas da sessão controlada. Abas pessoais preexistentes não são assumidas.
- **Pedidos ao Consecom:** caixa de tarefas no painel e histórico persistente. Uma fila local verifica a cada 10 segundos os pedidos de diagnóstico e versão predefinidos. Pedidos de linguagem natural ficam pendentes para execução pelo ChatGPT via MCP (necessita conversar com o agente); não existe integração com a conta ChatGPT para invocar este chat sozinho.
- 42 ferramentas MCP, total de seis novas sobre a versão anterior.

## Atualizações de um clique — como habilitar

1. Publique este repositório no **seu GitHub** e habilite Actions.
2. Ao criar uma tag `v2.6.1` para uma futura versão correspondente do código, o workflow `.github/workflows/publicar-release.yml` compila o instalador Windows e publica `consecom-update.json` com seu SHA-256.
3. Em **Consecom Plugin → Atualizações**, configure `dono/repositorio` e escolha se exige confirmação local.
4. Quando existir uma versão **mais nova** na release confiável, use **Atualizar agora**. O servidor pode reiniciar e o túnel reconectará.

**Importante:** esta versão precisa ser instalada inicialmente (ou publicada primeiro como release confiável), pois a versão anterior não conhece sua distribuição até que uma release esteja hospedada. Nenhum binário local/arquivo sandbox do ChatGPT pode ser baixado pelo atualizador sem uma release confiável.

## Fluxo inverso / agente local

- O painel aceita pedidos; o servidor guarda `task-inbox.json` no diretório privado de dados do Consecom.
- `tarefas_listar` e `tarefas_responder` possibilitam que este chat recupere e finalize pedidos quando o usuário pedir.
- O worker local revisa a fila a cada 10 segundos, executando **apenas** verificações internas predefinidas; não envia mensagens a um ChatGPT aberto nem executa pedidos arbitrários.
- Para execução **autônoma** de pedidos em linguagem natural, a próxima fase é instalar um trabalhador local que utilize a API Responses (com chave de API inserida pelo usuário no painel, cobrança independente do ChatGPT), ferramentas restritas, limites de custos e aprovações. Não há esse serviço nesta entrega.

## Limites de segurança

Acesso local: `http://127.0.0.1:8765`, com chave de sessão do painel. Sem terminal livre ou comando PowerShell arbitrário. Scripts NPM exigem aprovação local. O ChatGPT não acessa senhas/cookies e não deve publicar, enviar formulários ou comprar sem aprovação apropriada.

Para instalação veja [INSTALACAO_E_CONEXAO.md](INSTALACAO_E_CONEXAO.md).
