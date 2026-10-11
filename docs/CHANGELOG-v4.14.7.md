# Consecom v4.14.7 — recuperação da conexão Live

## Correções
- Removido o limite arbitrário de 50 mensagens por ciclo de vida do widget. A fila local e seus controles continuam aplicando limite de tamanho, cooldown e deduplicação.
- O widget mantém escuta com retentativa automática em falhas transitórias do MCP; a ausência temporária de `window.openai.callTool` não encerra definitivamente o laço.
- Reinicialização de polling em `pageshow` quando o navegador retoma a mesma página e em retorno de visibilidade, com proteção contra dois polls simultâneos.
- Preferência do checkbox de envio automático guardada pelo recurso `widgetState` quando disponível.
- Ao ativar recepção automática após um período passivo, o backend pode reivindicar mensagens antigas ainda em `queued`, mesmo quando o cursor do widget já passou. Reivindicação em lote é atômica e registrada antes do envio.
- Se o widget já exibiu uma mensagem anteriormente no modo passivo, ele ainda consegue enviar aquela mensagem **somente se** o fast-path recém-ativado provar que reivindicou seu ID.
- Mensagens em `claimed` sem confirmação de despacho por pelo menos 2 minutos aparecem no painel como possível envio interrompido. Reenvio só com clique e confirmação, pois o envio original pode ter sido aceito sem telemetria.

## Segurança
- Nunca reenvia automaticamente mensagens em `claimed`, `host_resolved` ou `answered`: isso pode duplicar turnos reais. O processamento de `queued` é seguro porque ainda não houve reivindicação.
- Sem novas credenciais, API paga da OpenAI, Work, extensão Chrome ou segundo MCP.
- Continuam válidas as aprovações no Windows e as regras de deduplicação de respostas.

## Limite da plataforma
Esta versão não torna a aplicação um serviço em segundo plano dentro do ChatGPT. A documentação do host não fornece garantia de polling infinito ou de execução automática quando a conversa está suspensa, recarregada, desconectada ou fechada. O servidor no Windows continua com a fila persistida, e o widget a drena quando volta a operar.

## Verificação
Testes Windows confirmam compilação, tenacidade da fila após cursor passivo, impossibilidade de duas reivindicações do mesmo evento, proteção de mensagens com entrega incerta, carregamento de estado e sintaxe JavaScript. É necessário um teste manual real após atualização para confirmar o comportamento do host.
