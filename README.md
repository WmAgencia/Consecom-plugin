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
- Catalogo em desenvolvimento: **72 ferramentas MCP**, sujeito a validacao no runner Windows e testes reais no PC.
- A release v4.0.0 permanece a versao distribuida ate merge, validacao e nova publicacao.
