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
