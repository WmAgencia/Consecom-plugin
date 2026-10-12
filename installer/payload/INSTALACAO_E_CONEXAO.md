# Consecom Plugin v4.15.0 — Conexao sem tunel

## Instalar e conectar
1. Execute **Consecom_Plugin_v4.15.0.exe** por cima da versao anterior. Nao desinstale: isso preserva suas pastas, permissoes e configuracoes.
2. Abra `http://127.0.0.1:8765/health` e verifique `version: 4.15.0`.
3. Na Central de Controle, clique em **Conectar ao ChatGPT**.
4. No Railway, projeto **consecom-mcp-cloud**, servico **consecom-mcp-relay**, copie o segredo `CONSECOM_LOCAL_WORKER_TOKEN` em **Variables**. Nao use a chave `x-api-key` do gateway de inferencia.
5. Cole o token no campo protegido do painel e marque **Ativar conexao automatica**. Aguarde **Railway conectado**. O computador faz conexoes de saida HTTPS; nenhuma porta publica nem tunel e necessaria.
6. No ChatGPT, use o plugin **Consecom** conectado ao MCP `https://consecom-mcp-relay-production.up.railway.app/mcp`. Se novas ferramentas nao aparecerem, atualize ou reconecte o plugin.

## Usar
- O Chat com ChatGPT do painel local envia mensagens para a fila persistente do Railway e recupera as respostas publicadas pelo MCP.
- As ferramentas `consecom_pc_status`, `consecom_pc_ferramentas` e `consecom_pc_executar` pertencem ao MCP do Railway. **Nesta primeira versao, ferramentas de computador remoto sao apenas de leitura e diagnostico**, seguindo os limites das pastas e das permissoes locais.
- O motor Ollama local de inferencia para Claude Code permanece **opcional** e nao e necessario para a comunicacao ChatGPT ↔ PC.
- O widget Consecom Live so encaminha mensagens de forma automatica enquanto o host ChatGPT mantiver o componente ativo; nao existe garantia de funcionamento com a conversa fechada.

## Migracao do tunel
- A partir desta versao, o Consecom **nao inicializa nem baixa tunnel-client**.
- O antigo instalador para o tunel, IDs `tunnel_...` e a chave da OpenAI Platform nao sao necessarios; os antigos endpoints locais de conexao do tunel respondem HTTP 410.
- Arquivos antigos de tunel nao sao apagados para permitir recuperacao local da instalacao anterior.

## Segurança
O relay instalado e de **uma unica conta proprietaria**, sem isolamento entre clientes. Nao compartilhe o token de worker, a API Key de teste, nem o endpoint publico com outros usuarios ate implementar autenticacao individual. Nao use o MCP em contextos nao confiaveis.
