# Consecom Plugin

Integração controlada entre o ChatGPT e um computador Windows por meio de um servidor MCP local e túnel privado.

**Versão publicada:** 4.0.0 · **Próxima versão em desenvolvimento:** 4.1.0

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

## Consecom Plugin v3.0.0

A v3 incorpora **19 novas ferramentas MCP**, totalizando 59:

- **Terminal de desenvolvimento:** iniciar `git status/diff/log`, `go test/build` e scripts `npm test/build/lint` em diretórios autorizados; consultar logs, cancelar e impor limite de 120 segundos. Executar scripts e builds exige aprovação no computador.
- **Git avançado:** validar patches de texto, solicitar aplicação com teste opcional e rollback automático quando o teste falhar, criar branches e reverter patches registrados; alterações exigem autorização local e respeitam o escopo das pastas.
- **Controle visual Windows:** consultar janelas visíveis, solicitar abertura de aplicativos permitidos, ativar, clicar, digitar e capturar uma janela, sempre com aprovação presencial. Capturas ficam guardadas temporariamente e expiram para consulta após 5 minutos.
- **Navegador 2.0:** configurar downloads em pasta privada, listá-los, solicitar cópia para projetos autorizados, selecionar arquivos para upload e confirmar cliques sensíveis (envio/publicação) localmente.

A versão foi compilada e submetida a testes automatizados no GitHub Actions. Recursos dependentes da interface interativa do Windows e de contas logadas **precisam ser validados no computador do usuário após a atualização**.

### Segurança
Não existe shell arbitrário, upload silencioso, acesso direto às credenciais ou automação irrestrita de compras/publicações. Scripts de projetos têm poderes próprios e exigem consentimento local. Antes de trabalhar em código importante, faça commit ou backup verificável.

### Atualizar
O repositório oficial é `WmAgencia/Consecom-plugin`; use **Atualizar** no menu esquerdo. O instalador verifica versão, SHA-256 e reinicia o servidor. Depois atualize o catálogo de ferramentas do plugin no ChatGPT para que as 59 operações apareçam.

## v3.0.1 — Aprovação máxima consistente

Quando o usuário habilita **Aprovação máxima** no próprio painel (confirmação `LIBERAR`), as operações permitidas das ferramentas v3 passam a executar diretamente, sem uma nova aprovação para cada ação: abrir e controlar janelas, capturar telas, builds/scripts autorizados, operações de Git e transferências/cliques no navegador. Nos modos **Sempre perguntar** e **Por categoria** continuam valendo os requisitos de aprovação anteriores.

**Importante:** Aprovação máxima elimina a aprovação local repetida, não as restrições técnicas. Comandos continuam limitados à lista permitida, Git/builds precisam de pasta de projeto autorizada, áreas de sistema/credenciais e caminhos perigosos continuam bloqueados, e o navegador só controla a sessão selecionada. A política independente de atualização remota do programa não é alterada.

## Consecom v4.0.0 — Primeira etapa

10 novas ferramentas MCP (69 no total):
- **Editor de codigo**: buscar termos em ate 250 arquivos de projetos autorizados; ler ate 160 linhas com SHA-256; substituir trecho unico com verificacao de versao e backup.
- **Windows visual**: descobrir controles Win32 de uma janela e clicar por ID real, respeitando modo de aprovacao.
- **Criatividade**: detectar Blender, FFmpeg, FFprobe, ImageMagick, Python e Node; inspecionar metadados de videos com FFprobe. **Ainda nao edita nem renderiza videos automaticamente.**
- **Fluxos de tarefas**: executar ate 6 comandos de desenvolvimento predefinidos sequencialmente, interrompendo se um falhar. Acompanhar etapa, logs e cancelar. O fluxo nao e um agente IA independente de conversas.

**Seguranca:** ferramentas limitadas a pastas autorizadas e listas de comandos, sem shell arbitrario, captura de credenciais ou elevacao silenciosa. Acesso administrativo futuro deve usar UAC, registros e escopos especificos.


## V4.1.0 em desenvolvimento (nao publicada)
- `codigo_previsualizar_diff`: previa textual de uma substituicao exata, com verificacao de versao SHA-256 sem tocar no arquivo.
- `codigo_restaurar_backup`: restauro de snapshot criado pelas edicoes desta fase, bloqueado se houve modificacao externa, com backup preventivo para recuperacao.
- `fluxo_historico`: ultimos 20 fluxos com progresso e status; snapshots locais gravados durante cada etapa e recuperados apos reinicio. Fluxos interrompidos **nao** sao retomados automaticamente.
- Catalogo em desenvolvimento: **78 ferramentas MCP**, sujeito a validacao no runner Windows e testes reais no PC.
- A release v4.0.0 permanece a versao distribuida ate merge, validacao e nova publicacao.


### Adicoes desta iteracao (pendentes de validacao no PC)
- `video_extrair_frame`: JPEG limitado por frame via FFmpeg em timestamp exato dentro de pasta autorizada.
- `video_diagnosticar`: heuristicas de preto/congelamento com ate 20 s de amostra (resultados podem ser cenas intencionais).
- `video_render_iniciar`, `video_render_status`, `video_render_cancelar`: exportacao assincrona MP4 H.264/AAC com tres proporcoes, tempo maximo 180 s por tarefa e saida em nova pasta `consecom-renders` (precisa escrita autorizada).
- `codigo_formatar_go`: formatacao gofmt com SHA256 esperado, permissao de escrita, backup recuperavel e aprovacao conforme modo local.
- Deteccao criativa agora prova o executavel e versao antes de considerar instalado e ignora aliases de loja WindowsApps para Python.
- Codigo de base UI Automation Windows adicionado sem registro MCP de novas acoes; **ainda nao entregue como recurso funcional**.
- Testes Windows e artefato PREVIEW dependem do CI da branch; nenhuma nova release deve ser declarada sem o resultado final.


### Correcao 4.1.1 — consistencia da Aprovacao Maxima

- `dev_execucao` passa a executar os comandos npm permitidos diretamente no **modo maximo**, sem adicionar aprovacao local ao Consecom. Em modos `always` e `category`, scripts continuam exigindo confirmacao no painel.
- `politica_aprovacao` informa o modo efetivo, se prompts locais foram dispensados, quantas pastas de projeto estao autorizadas e o estado separado de atualizacoes remotas.
- A descricao das ferramentas de fechar processos, plano de energia e gravacao foi alinhada ao comportamento real do modo maximo.
- Aprovacao maxima e uma preferencia **do servidor local Consecom**, nao uma permissao da plataforma ChatGPT nem do Windows: bloqueios do cliente MCP, UAC, filtros de pastas protegidas, listas de comandos permitidos, limites de tamanho e verificacoes de SHA permanecem.
- Comandos com `root_id` continuam exigindo cadastro explicito da pasta no painel; nao existe elevacao implicita do acesso ao projeto.
- `atualizacao_solicitar` permanece protegida por uma permissao **independente**, `remote_approved`, escolhida no proprio painel. Nao se habilita atualizacao remota de executaveis apenas por usar o modo maximo.
- Selecionar um navegador/perfil ainda e uma preferencia explicita e separada; nao se assume o perfil pessoal automaticamente.
- Se o ChatGPT bloquear uma acao no cliente antes que ela chegue ao MCP, a alteracao do Consecom nao tem como remover esse bloqueio.

- Ao ativar o modo maximo, pedidos antigos de aprovacao local de outros modos sao **cancelados, jamais executados automaticamente**. Reenvie a acao que deseja executar. Um pedido de atualizacao remota permanece pendente por usar uma permissao separada.


### 4.2.0 — Abrir aplicativos do Windows sem lista fixa

O Consecom possui tres novas ferramentas MCP:

- \`desktop_aplicativos\`: pesquisa nomes e IDs de aplicativos registrados no menu Iniciar (inclusive apps da Microsoft Store, como **Relogio**). Informe \`pesquisa\` ou deixe vazia para exibir ate 120 resultados.
- \`desktop_abrir_instalado\`: abre um dos aplicativos retornados pelo seu \`app_id\` exato, usando o mecanismo nativo \`shell:AppsFolder\` do Windows, sem argumentos extras. Caso haja nomes semelhantes, escolha pelo ID descoberto, nunca por suposicao.
- \`desktop_abrir_executavel\`: abre um arquivo \`.exe\` local existente por caminho absoluto, tambem sem parametros. Cobre programas portateis nao registrados no menu Iniciar. Atalhos \`.lnk\`, scripts, arquivos de lote e caminhos de rede nao sao executados por essa ferramenta.

Em **Aprovacao Maxima**, ambas as ferramentas de abertura executam diretamente, sem gerar nova aprovacao no painel Consecom. Nos demais modos, continuam exigindo a confirmacao local normal. O modo maximo nao contorna autorizacoes do ChatGPT, Windows/UAC, protecoes de pastas do sistema nem permite iniciar com privilegios elevados silenciosamente.

Fluxo recomendado: procurar \`Relogio\` em \`desktop_aplicativos\`, escolher o \`app_id\` real, chamar \`desktop_abrir_instalado\`, verificar a janela com \`desktop_janelas\` e usar controle visual para configurar um alarme. **Abrir o aplicativo nao significa que um alarme foi criado**: confirme a configuracao na interface. Alarmes de aplicativos podem ser recorrentes; uma ocorrencia em dia especifico requer verificar o comportamento do Relogio instalado.

Limites: a lista reflete os aplicativos que o perfil Windows atual registra no menu Iniciar; executaveis adicionais podem ser abertos pelo caminho absoluto. Aplicativos do sistema protegidos, comandos arbitrarios, scripts e elevacao UAC automatica continuam vedados.


### 4.3.0 — Central de desenvolvimento: GitHub, Vercel, Supabase e Railway

**Fluxo recomendado no ChatGPT:** Consecom (arquivos/terminal/build local) + conectores oficiais GitHub/Vercel/Supabase/Railway (projetos/deploy/banco/servicos). Os conectores sao independentes: a permissao OAuth no ChatGPT **nao** transfere tokens para o executavel do Windows. Nao e necessario copiar uma API key para o Consecom para usar essas quatro nuvens **atraves do ChatGPT**.

Novas ferramentas MCP locais:
- \`dev_integracoes_status\` — indica se Git, GitHub CLI (\`gh\`), Node.js, npm, Vercel CLI, Supabase CLI, Railway CLI e Go podem ser localizados no Windows, sem executar comandos, ler tokens ou verificar credenciais. Lista tambem quantas pastas de projeto foram autorizadas pelo usuario.
- \`dev_projeto_inspecionar\` — dado um \`root_id\` ja aprovado no painel, verifica \`package.json\`, detecta framework (Next.js, Vite, Astro, React, Vue, Nuxt, Remix, SvelteKit, HTML) e informa **apenas os nomes** dos scripts npm e a presenca de arquivos de configuracao Vercel, Supabase, Railway e Git. Nunca le \`.env*\`, arquivos de credenciais ou \`.git/config\`; tambem nao executa scripts.

Passos para desenvolver um site:
1. Cadastre a pasta especifica do projeto no painel local Consecom, com escrita habilitada somente se precisar editar. Aprovacao Maxima nao adiciona pastas automaticamente.
2. Rode \`dev_integracoes_status\`, \`dev_projeto_inspecionar\`, edicoes com backup e \`dev_job_iniciar\`/fluxos de build/teste.
3. Use o conector GitHub do ChatGPT para criar branch, commits e PRs no repositorio selecionado.
4. Use o conector Vercel para vincular/deployar, configurar dominios e ver logs no escopo certo.
5. Use o conector Supabase para tabelas, SQL, migrations e edge functions do projeto escolhido, com revisao de alteracoes destrutivas.
6. Use o conector Railway para projetos, servicos, ambientes, variaveis e deployments quando precisar de backend/worker.

O plugin **nao** promete acesso irrestrito a contas desconhecidas, nao le nem salva tokens dos provedores, nao executa \`git push\` ou um deploy sem fluxo explicitamente solicitado e nao substitui permissoes dos conectores, escopos de equipe, autenticacao e UAC. Para uma execucao local independente do ChatGPT, autentique as CLIs em seu proprio PC pelos fluxos oficiais dos provedores; a verificacao de instalacao nao significa sessao autenticada.


### 4.4.0 PREVIEW — Servidor de desenvolvimento local persistente

Cinco ferramentas MCP em preview: `dev_preview_iniciar`, `dev_preview_status`, `dev_preview_listar`, `dev_preview_parar`, `dev_preview_verificar`.

- O usuario cadastra uma **pasta especifica** de projeto Node com permissao de escrita no painel Consecom e instala dependencias (`node_modules`). O software nao instala pacotes automaticamente.
- So aceita projetos com `package.json`, script `dev` e dependencia identificada como Next.js, Vite ou Astro. O script pode executar codigo arbitrario do **proprio projeto**; somente pastas cadastradas executam codigo.
- Inicia `npm run dev` com flags fixas (framework conhecido) e host `127.0.0.1` / porta selecionada entre 1024 e 65535. Nenhum comando ou caminho arbitrario entra no shell.
- Limite de **2 servidores simultaneos**. Sessao de no maximo **60 minutos**; encerra processo e descendentes pelo Windows. Os previews sao **efemeros**: nao sobrevivem a reinicio do Consecom.
- Nos modos com aprovacao local, o primeiro start exige aprovacao no painel; em modo `maximum`, executa sem novo prompt local.
- `dev_preview_verificar` faz apenas um GET HTTP no URL loopback criado pela propria sessao, com 4 s de timeout, limite de 1 MiB, sem seguir redirects, retornando status/Content-Type/presenca de title, viewport, description e h1. **Nao e um teste Lighthouse ou Playwright.**
- Nao le arquivos `.env`, nem devolve logs brutos de processo ao ChatGPT (scripts do projeto podem emitir segredos). Para inspecionar erros, use testes de build/lint aprovados e logs locais sob seu controle.

**Atenção:** esta etapa e um prototipo de ferramenta para desenvolvimento pessoal, nao uma promessa de servidores publicos em producao. Testar no Windows real antes de colocar em planos pagos; verificar politica de encerramento de subprocessos em casos de falha/reinicio do serviço.


### 4.5.0 — Visual QA + ciclo de reparo assistido + preflight cloud

Esta versao evolui as tres etapas em cima do servidor de previa v4.4.0. Ferramentas novas:
- **Inspecao visual:** \`browser_qa_auditar\` devolve problemas DOM estruturais de SEO/acessibilidade/overflow/imagens. \`browser_qa_responsivo\` captura mobile/tablet/desktop via Chrome DevTools Protocol e **restaura viewport**. \`browser_qa_instrumentar\` registra apenas categorias de console.error, erros e promessas rejeitadas na aba controlada; \`browser_qa_erros\` devolve contagens. A captura visual requer aba criada/controlada pelo Consecom; nao acessa outras abas. A coleta de categorias de erros deve ser ativada **antes** das interacoes a observar. Screenshot pode conter dados pessoais visiveis.
- **Ciclo de reparo:** \`dev_reparo_planejar\` identifica os scripts \`lint\`, \`test\`, \`build\` de \`package.json\` ou comandos Go. \`dev_reparo_iniciar\` executa sequencia de verificacoes por \`fluxo_iniciar\`, com monitoramento por \`fluxo_status\`. O ChatGPT interpreta as falhas e prepara diffs de codigo; \`git_patch_validar\`/\`git_patch_aplicar\` verificam o patch. Novo teste \`npm_quality\` executa lint/test/build e **reverte automaticamente o patch** se qualquer teste falhar. Nenhum modelo AI escondido executa arbitrariamente codigo ou gera patches sem ChatGPT.
- **Vinculos para deploy:** \`dev_servicos_associar\` armazena apenas identificadores de repositorio GitHub, projeto/equipe Vercel, ref Supabase, projeto Railway e ambiente do workspace. \`dev_servicos_inspecionar\` consulta a associacao e \`dev_deploy_preflight\` verifica pasta autorizada, Git status limpo e destinos definidos. Os IDs fornecidos **nao sao validados remotamente pelo plugin**: o ChatGPT deve verificá-los pelos conectores GitHub/Vercel/Supabase/Railway e usar seus respectivos escopos para deploy. Nenhuma chave, token ou .env e lida, persistida ou transferida. O Consecom **nao executa deploy em nuvem autonomamente** e o ambiente production exige revisao independente.

Limites atuais e proximos passos: Browser QA e baseado em CDP e nao substitui Lighthouse, axe-core nem suite Playwright completa. Pipeline de reparo e assistido pelo ChatGPT, nao um agente LLM independente rodando em background. Vinculos cloud sao referencias e preflight local, nao autenticação ou deploy; os conectores oficiais realizam as acoes depois que o ChatGPT valida os destinos. Testes de integracao no computador real exigem cadastrar um projeto temporario no painel local. Sem isso, apenas os testes do GitHub Actions Windows estao disponiveis.


### v4.6.0 — Integracao opcional com agent-browser (Vercel Labs)

**Fonte oficial**: https://github.com/vercel-labs/agent-browser (licenca Apache-2.0; software de terceiro, nao incluido no instalador Consecom). O link https://github.com/topics/agentic-browser e uma *colecao* de repositorios, nao uma ferramenta instalavel. Foi escolhido o \`vercel-labs/agent-browser\` por oferecer CLI de navegacao com arvore de acessibilidade e referencias de elementos.

Instalacao local da CLI no Windows (necessita Node.js e npm compativeis com a versao da CLI):

\`\`\`powershell
npm install -g agent-browser
agent-browser install
\`\`\`

O segundo comando baixa o Chrome for Testing necessario quando nao existe um navegador adequado. Nenhum desses comandos e executado automaticamente pelo Consecom. O motor externo usa memoria/CPU/disco do cliente.

Novas ferramentas:
- \`navegador_agente_status\`: detecta a CLI opcional e indica estado da sessao.
- \`navegador_agente_abrir\`: cria uma sessao independente de Chromium, visivel ("headed"), com dominio limitado e sem copiar credenciais/perfil pessoal. Aceita HTTPS e HTTP local explicitamente em 127.0.0.1 ou localhost (porta obrigatoria para localhost). Nao aceita IPs de rede privada, URL com usuario/senha, query string ou fragmento.
- \`navegador_agente_ler\`: obtem accessibility snapshot apenas da aba do Consecom, com refs \`@eN\`.
- \`navegador_agente_clicar_link\`: somente em links da ultima captura. Requer ref valida e barra links com rotulos de pagamento, exclusao, publicacao, envio etc. Botoes e submit devem usar fluxo de aprovacao independente.
- \`navegador_agente_preencher\`: apenas textbox/searchbox/input da ultima captura; campos de senha, cartao e token bloqueados. Nao envia formularios.
- \`navegador_agente_capturar\`: faz screenshot PNG efemero e retorna base64 com limite de 1,1 MiB.
- \`navegador_agente_fechar\`: fecha somente a sessao isolada criada pelo Consecom.

Limites: o mecanismo de navegacao inteligente utiliza o **ChatGPT para decidir quais refs clicar**, nao inclui IA da Vercel nem executa a funcao \`chat\` externa do agent-browser. Cada sessao e restrita ao dominio indicado; CDNs externos e logins cross-domain podem ser bloqueados, intencionalmente. O plugin nao tenta assumir sua sessao Chrome atual. Website text e conteudo externo sao dados nao confiaveis, nunca instrucoes para executar comandos. O modo Aprovação Máxima nao desbloqueia pagamentos, exclusoes ou senhas.

Verificacao: execute \`navegador_agente_status\`; apos instalar a CLI, tente \`navegador_agente_abrir\` com \`https://example.com\`, \`navegador_agente_ler\`, \`navegador_agente_capturar\`, \`navegador_agente_fechar\`. O Windows CI do repositorio compila e valida os filtros/escopos; uma prova de navegacao real exige a CLI instalada na maquina ou um smoke-test adicional com Chrome em CI.
