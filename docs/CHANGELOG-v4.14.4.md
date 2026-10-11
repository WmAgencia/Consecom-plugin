# Consecom 4.14.4 — Indicador de digitação e teste real 10x

## UX do chat

Após o Consecom **registrar um despacho real** de uma mensagem para o ChatGPT (`followup_at` ou `host_resolved`), o painel exibe uma bolha “ChatGPT está digitando” com três pontos animados no espaço de resposta. O indicador desaparece quando houver `consecom_chat_responder(tipo=final)` persistido.

- Uma mensagem somente em `queued` ou `claimed` **não** exibe o indicador, pois ainda não há prova de despacho.
- A animação é um **indicador visual de espera**, não streaming de tokens nem prova de que o modelo iniciou a geração. Essa ressalva fica no atributo de ajuda do elemento.
- Mensagens finais removem a bolha e exibem o texto persistido, sem imitar uma resposta.
- Acessibilidade: marcador de status sem repetição desnecessária e `prefers-reduced-motion`.

## Ensaio automatizado de dez mensagens reais

O botão `▶ Testar 10x` no chat local permite iniciar **dez mensagens reais**, sequenciais, usando o Consecom Live já conectado ao ChatGPT Chat. A rotina:

1. Confirma recepção automática ativa e ausência de mensagens anteriores ainda em trânsito.
2. Exige confirmação explícita antes de enviar dez solicitações reais.
3. Publica uma mensagem identificada por rodada no painel pelo endpoint autenticado.
4. Consulta o diagnóstico local para medir `sent_at → followup_at`, sem incluir tempo de geração do modelo.
5. Espera uma **resposta final verdadeira** antes de enviar a próxima, evitando concorrência de dez turnos ao ChatGPT.
6. Respeita intervalo local de envio (3,4 s), interrompe se a aba ficar oculta ou houver recusa e mostra progresso.
7. Se a resposta não chegar em 120 s, interrompe com resultado parcial (não inventa dados).
8. Ao final apresenta média, mediana, P90, pior e quantidade acima de 15s dos **tempos exatos**, discriminando os casos que só tenham um limite superior por faltar o timestamp do despacho.

Essa rotina **não funciona autonomamente quando o ChatGPT suspende o widget ou quando a aba de controle está fechada**. Ela não utiliza OpenAI API, Work, segundo MCP, bots de sessão ou automação do navegador pessoal.

## Dez simulações automáticas executadas no CI

O teste `TestTenAutomaticLocalBridgeCyclesWithoutOpenAI` executa dez rodadas isoladas da fila, `waitAndClaim`, log de despacho, aceitação simulada, publicação de resposta simulada, exibição e persistência. Ele registra os dez tempos em log do CI. Isso **não mede velocidade real do ChatGPT**. O teste real do botão precisa ocorrer no PC do usuário com Live ativo.

## Critério de aceitação

Objetivo: tempo de envio → despacho <=10 s, limite de referência <=15 s. Fazer no mínimo 10 rodadas reais, com todas as medições exatas disponíveis, e analisar mediana, P90 e exceções. Testes Go ou 1 amostra real isolada não justificam declarar essa meta atingida com consistência.
