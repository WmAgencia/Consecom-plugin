# Consecom Plugin 2.6 — instalação e conexão

1. Instale a atualização por cima da 2.5. **Não desinstale a versão existente**. O arquivo de configuração, o túnel oficial e perfis de navegador ficam preservados.
2. Abra `http://127.0.0.1:8765`; verifique a versão **2.6.0** e o estado do túnel.
3. No ChatGPT, atualize a lista de ferramentas do plugin. São esperadas **42 ferramentas**.
4. Clique em **Atualizar agora** para verificar a release configurada. Se ainda não houver repositório confiável, selecione **Configurar** e informe `dono/repositorio` depois de publicar a release.
5. Para o terminal, cadastre a pasta do seu projeto em **Pastas autorizadas**; use `dev_execucao` via ChatGPT. Para rodar `npm test/build/lint`, abra o painel local e aprove o comando específico.
6. No painel, escreva pedidos em **Pedidos ao Consecom**. Diagnóstico e versão são processados automaticamente a cada 10 s. Pedidos livres aguardam o atendimento do ChatGPT via `tarefas_listar` e `tarefas_responder`; esse chat **não** recebe notificações instantâneas da fila.

O chat no celular consegue chamar as ferramentas remotas sempre que o plugin estiver ativo, PC e túnel conectados. Um agente 24h autônomo exige integração de API externa e controle próprio de execução/custos — não está incluído nesta versão.

## Atualizações oficiais desde v2.6.1

A origem oficial está fixa em `WmAgencia/Consecom-plugin`; não é necessário cadastrar repositório no painel. O botão **Atualizar** fica no canto inferior esquerdo, acima de **Servidor MCP**. Um clique verifica a versão, baixa o instalador, valida seu SHA-256 e inicia a instalação quando houver versão mais recente. Atualizações iniciadas remotamente pelo ChatGPT mantêm a política de aprovação local, salvo autorização prévia explícita. Para migrar de uma versão anterior que ainda exige configuração de repositório, instale a v2.6.1 uma vez pelo GitHub Releases.
