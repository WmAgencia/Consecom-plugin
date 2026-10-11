# Consecom Plugin v4.10.0 — Contexto Graph View

## Interface
A página **Contexto** agora tem apenas:
- Cartões de integração no topo, com ChatGPT em primeiro.
- Um grafo grande e navegável na área restante.

O grafo inclui pan em espaço vazio, zoom no cursor com roda do mouse,
arrastar nós individuais com o ponteiro, destaque de vizinhos, busca de nós,
menu de contexto (botão direito), prévia de fontes/notas e ajustes de
repulsão e distância. O botão Centralizar restaura a visão.

As fontes cadastradas são representadas por **pastas lógicas** em nós
coloridos; notas cadastradas aparecem sob a fonte, com vínculos da lista
\`links\`. O cofre Obsidian aparece como nó central quando conectado.

**Sem dados fabricados.** Encontrar o executável Claude Code/OpenCode ou
registrar a fonte MCP não importa automaticamente o histórico desses
aplicativos. A versão atual mostra as fontes e notas já registradas pelo
Consecom, não lê indiscriminadamente o cofre inteiro.

## Segurança
- Prévia textual utiliza \`textContent\` e as APIs locais autenticadas.
- Arrastar um nó afeta somente o layout visual; não move arquivos do disco.
- Conexões anteriores e configurações locais são preservadas.
- Sem login oculto no ChatGPT; sem API paga e sem ChatGPT Work.
- Visualização inspirada nas funcionalidades do Graph View do Obsidian,
  com agrupamento adicional por pasta lógica, sem reproduzir sua marca.

## Testar
Após instalar, abra \`http://127.0.0.1:8765/#contexto\` e confirme:
1. Só há os cartões de integração e um grafo; os painéis antigos desapareceram.
2. Pressione um nó e arraste. Outras bolinhas não devem ir junto.
3. Arraste o fundo; use a roda para zoom no ponteiro.
4. Passe o cursor em um nó: as ligações vizinhas ganham destaque.
5. Clique em uma nota real para ver sua prévia; botão direito abre ações.
6. Ajuste a força e distância pelo ícone de engrenagem.
7. Não devem ser exibidas notas ou histórias inexistentes.
