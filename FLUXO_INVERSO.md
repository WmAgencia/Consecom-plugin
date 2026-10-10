# Fluxo inverso — do painel para a IA

## O que esta versão entrega

- Caixa de entrada com solicitações tipadas e histórico em disco.
- Polling **local** a cada 10 segundos. Apenas pedidos de diagnóstico e versão são executados de forma autônoma.
- Ferramentas MCP para listar e registrar os resultados de tarefas livres quando o ChatGPT é acionado pelo usuário.

## Arquitetura recomendada para o agente persistente

Consecom (painel) -> fila durável -> worker local -> OpenAI Responses API / modelo escolhido -> planejamento -> ferramentas com escopo -> validação -> aprovação para ações críticas -> relatório no painel.

Regras: chave API inserida e protegida no Windows, não enviada na conversa; orçamento e limites de iterações; um job por projeto para alterações conflitantes; snapshots e reversão; logs redigidos; sem publicar/enviar/excluir sem autorização específica. A API é cobrada separadamente da assinatura ChatGPT.

Não existe mecanismo no MCP para forçar o ChatGPT desta conversa a ler uma fila automaticamente a cada 10 segundos. `automations`/tarefas agendadas do ChatGPT também não substituem esse worker local de 10 segundos.