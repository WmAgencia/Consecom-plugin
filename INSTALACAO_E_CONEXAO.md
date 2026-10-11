# Consecom Plugin — instalação 2.7.2

## Atualizar pelo painel

1. Abra `http://127.0.0.1:8765` no Windows.
2. Clique em **Atualizar**, na parte inferior da barra lateral, acima de **Servidor MCP**.
3. O programa consulta a release oficial de `WmAgencia/Consecom-plugin`, valida versão e SHA-256, baixa o instalador e executa a atualização.
4. Aguarde o servidor e o túnel reiniciarem. A partir da v2.7.2, a página verifica o reinício e recarrega automaticamente. Ao migrar de versões anteriores, use Ctrl+Shift+R caso a página não se atualize.
5. No ChatGPT, atualize o catálogo do plugin MCP se ferramentas antigas ainda aparecerem. Esta versão apresenta **40 ferramentas**.

## Primeira instalação

Utilize o instalador da [última release oficial](https://github.com/WmAgencia/Consecom-plugin/releases/latest), mantendo a instalação existente. Não desinstale para atualizar. A configuração e as credenciais criptografadas do túnel são preservadas.

## Estrutura do painel

- **Painel:** métricas, discos, políticas de aprovação, navegador e permissões de pastas.
- **Laboratório de Agentes:** registros e atividades de agentes conectados.
- **Atualizar:** botão lateral fixo com origem GitHub já predefinida.

A interface de conversa e o formulário de mensagens foram removidos nesta versão. O histórico local antigo não é excluído.

## Segurança

Acesso local: `127.0.0.1:8765`. Scripts npm exigem aprovação. O controle do navegador tem bloqueios para ações sensíveis; nunca informe senhas pelo chat. O ChatGPT não fica consultando a máquina por conta própria sem a conversa ativa.
