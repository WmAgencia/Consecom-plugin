# Consecom v4.14.5 — Chat mais simples

## Alterações
- Removido o botão **Testar 10x** da Central de Controle.
- Removidos script, estado, contador, estilos e testes automáticos específicos desse ensaio.
- Mantido o envio manual de mensagens, histórico bidirecional, diagnóstico de latência e a bolha animada **ChatGPT está digitando** após o despacho ser registrado.
- Avisos de mensagem na fila e aguardando resposta mantêm prioridade correta na barra de estado.
- Testes compactos agora conferem que a interface não contém referências ao benchmark, que a animação só corresponde a mensagem despachada, que a resposta final encerra a espera, e que o JavaScript do painel compila corretamente.

## Inalterado
- Um único MCP Consecom, sem OpenAI API, Work ou outro serviço.
- Claim atômico, deduplicação e telemetria do host.
- Atualização por release assinada com SHA-256 e confirmação de permissões locais.
- Testes de segurança, regressão e integração já existentes.

## Validação
Compilação Windows, Go tests e verificação de sintaxe JavaScript são executados na CI.
O usuário fará os testes **reais de latência manualmente**. A interface não enviará perguntas de teste em sequência.
