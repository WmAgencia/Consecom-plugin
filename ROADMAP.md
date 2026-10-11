# Roadmap de extensões úteis ao Consecom MCP

## Prioridade alta
1. **Diagnóstico/execução de tarefas do projeto**: executar npm test, npm run build, go test e comandos estritamente permitidos por projeto, com timeout, limites de CPU/RAM, logs e aprovação.
2. **Editor de código com patch/diff e Git**: leitura em lotes, busca textual, diffs, commits locais, branches, restauração de versões. Necessário para grandes alterações de código que hoje excedem 32 KiB por arquivo.
3. **Operações avançadas de navegador**: múltiplas abas, navegação por elementos, screenshots, downloads autorizados, controle de sessões e confirmação explícita para formulários.
4. **Visão do desktop e UI Automation**: abrir aplicações permitidas, identificar botões/janelas e clicar através da API de acessibilidade do Windows com controles de segurança.
5. **Gerenciador de processos autorizados**: iniciar/parar serviços de desenvolvimento permitidos e consultar logs sem abrir shell arbitrário.

## Prioridade média
6. **Agendador local**: tarefas em horário configurado, relatórios e notificações; execução continua sem ChatGPT aberto.
7. **GitHub/GitHub Actions**: PR, CI, deploy e rollback com limites por repositório.
8. **Uploads e downloads com controle**: anexos, CSV, PDFs e exportações em diretórios autorizados.
9. **Laboratório multiagente real**: workers persistentes, filas, locks por arquivo, revisão crítica, estados e logs reais.
10. **Gerenciamento de backup corporativo**: snapshots de diretórios de projeto, retenção e backup offsite criptografado.

## Prioridade opcional / segurança maior
11. **Observabilidade de projetos e servidores**: Railway/Vercel/Supabase, saúde, erros, métricas e notificações.
12. **Controle de aplicações desktop**: Excel/Word/Figma/VS Code por adaptadores dedicados, não mouse cego.
13. **Automação de publicidade da agência**: relatórios, drafts de anúncios, CRM, revisão de leads e execução autorizada em plataformas.
14. **Acesso por celular**: tarefas assíncronas, fila de trabalhos, aprovações remotas autenticadas e avisos de finalização.
15. **Administração do Windows supervisionada**: Wi‑Fi, energia, drivers e atualizações de sistema apenas com confirmação presencial; nunca credenciais/segredos.

As funcionalidades desta lista são propostas, não estão implementadas no Consecom 2.5.

## Fases seguintes a v4.0.0
- Controle visual por Microsoft UI Automation (UIA) de apps modernos, com arvore semantica, foco, estados e formularios.
- Estudio de videos: Remotion/FFmpeg, verificacao de frames, transcricoes, legendas, motion e exportacao.
- Integracao Blender Python/render local se o Blender estiver instalado.
- Integracao com editores de imagem e projetos em camadas: GIMP/Krita/Inkscape quando disponiveis.
- Motor de tarefas com checkpoints, dependencia, recuperacao, agendamentos e execucao independente de chat via servico local configurado.
- Administracao elevada **por UAC** para tarefas privilegiadas delimitadas, auditadas e com desligamento de emergencia, sem burlar protecoes Windows.
- Recursos adicionais: busca global por escopo, instalador WinGet, diagnosticos de rede, monitoramento, Office e backup.


## Plano de entrega verificavel (apos v4.1.0)

O desenvolvimento deve manter `main` e o atualizador oficial estaveis enquanto cada fase passa no CI Windows e, para UI/video/UAC, em testes locais reais. Contagem do catalogo nao equivale a funcionalidade pronta.

### Fase 1: Estudio FFmpeg (iniciada no PR #5)
- Entrega em progresso: inspecao por FFprobe, frame JPEG, diagnostico de cenas escuras/congeladas, render MP4 assincrono H.264/AAC com tres formatos, cancelamento e logs.
- Aceite: teste de arquivo MP4 real com FFmpeg no Windows, validacao do arquivo exportado via FFprobe e teste em projeto do computador cadastrado pelo usuario. Render nao deve sobrescrever originais.
- Pendente: render multi-cena, tratamento de audio detalhado, estilos de legenda, motion, edicao baseada em timeline e QC automatizado estendido.

### Fase 2: UI Automation Windows moderna
- Base Windows UIA ja existe, mas **nao esta exposta via MCP**; integracao operacional depende de revisao de seguranca e permissao da ferramenta.
- Aceite: inspecao sem revelar campos de senha, localizacao estavel por automation_id e control_type, protecao de janelas sensiveis, focus/invoke, teste no Notepad e app moderno, logs e cancelamento.
- Recursos seguintes: ScrollPattern, SelectionPattern, ValuePattern protegido, Drag&Drop, verificacao antes/depois com screenshot autorizado.
- Nao interagir com janelas de credenciais, UAC ou logon.

### Fase 3: Remotion e Blender
- Detectar a presenca real de executaveis e de dependencias por projeto; nao presumir que Node significa Remotion.
- Remotion: template de composicao de projeto autorizado, calculo de frames, render/preview com timeout, validacao de MP4 com FFprobe, retries com teto.
- Blender: por CLI headless, `--disable-autoexec`, arquivos e outputs na pasta autorizada, limites de render e cancelamento; rejeitar scripts e .blend nao confiaveis.
- Aceite: projeto de exemplo minimo testado em Windows com binarios instalados; ausencia de binario deve retornar erro claro sem fingir render concluido.

### Fase 4: Orquestracao autonoma persistente
- Historico de fluxos existente, mas reinicio registra `interrupted`; **nao existe retomada autonoma**.
- Adicionar fila local autenticada, dependencias, checkpoints idempotentes, tentativa de novo com limite e backoff, rollback e shutdown seguro. Nenhum fluxo deve iniciar sem escopo e permissao definidos.
- Aceite: simulacao de reinicio, cancelamento durante uma etapa, reexecucao idempotente e comprovacao de que a fila nao reexecuta efeitos irreversiveis.

### Fase 5: Administracao supervisionada e extensoes
- Helper UAC separado com tarefas allowlisted e parametros tipados, privilegio por duracao minima, eventos auditados, sem bypass de prompts do Windows.
- Diagnostico de rede e processo read-only primeiro; alteracoes privilegiadas apenas por helper aprovado pelo Windows.
- Docker/WSL, Office/PDF e deploy: adaptadores delimitados, checagem de instalacao, versoes suportadas, backup e testes por recurso.

### Gates obrigatorios de release
1. Branch de desenvolvimento e PR em draft ate testes positivos em todos os sistemas afetados.
2. `go build`, `go test ./...` e validacao Windows no HEAD exato; testes de integracao de binarios locais quando disponiveis.
3. Instalador PREVIEW isolado do release oficial, acompanhado de SHA-256; testar instalacao e rollback.
4. So fazer merge e publicar a proxima versao apos aprovacao da validacao no computador e confirmacao dos artefatos `.exe` e `consecom-update.json`.
