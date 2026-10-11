package main

import (
 "strconv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	Annotations map[string]any `json:"annotations"`
}

func schema(fields map[string]any, required ...string) map[string]any {
	// MCP inputSchema follows JSON Schema: "required" must be a string array,
	// and must never be JSON null. Omit it for tools without required fields.
	out := map[string]any{"type": "object", "properties": fields, "additionalProperties": false}
	if len(required) != 0 {
		out["required"] = required
	}
	return out
}
func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }
func tools() []Tool {
	readonly := map[string]any{"readOnlyHint": true}
	mutating := map[string]any{"readOnlyHint": false, "destructiveHint": false, "openWorldHint": false}
	return []Tool{
		{"diagnostico_pc", "Ler RAM total/disponivel, uso de CPU e data do computador, sem alterar configuracoes.", schema(map[string]any{}), readonly},
		{"listar_processos", "Listar processos com maior consumo de memoria; apenas diagnostico.", schema(map[string]any{}), readonly},
		{"pastas_autorizadas", "Listar apenas pastas de projeto autorizadas pelo usuario no painel local.", schema(map[string]any{}), readonly},
		{"listar_arquivos", "Listar ate 120 arquivos de texto e diretorios de uma pasta autorizada, sem caminhos confidenciais.", schema(map[string]any{"root_id": str("ID da pasta autorizada"), "caminho": str("Caminho relativo, como src ou src/components")}, "root_id", "caminho"), readonly},
		{"ler_arquivo", "Ler somente arquivo UTF-8 de ate 64 KiB na pasta autorizada; segredos e arquivos ocultos sensiveis bloqueados.", schema(map[string]any{"root_id": str("ID da pasta autorizada"), "caminho": str("Caminho relativo do arquivo")}, "root_id", "caminho"), readonly},
		{"terminal_restrito", "Executar somente comandos diagnosticos preaprovados: windows_version, git_status, git_branch, git_log; sem interpretador de comandos livre.", schema(map[string]any{"acao": map[string]any{"type": "string", "enum": []string{"windows_version", "git_status", "git_branch", "git_log"}}, "root_id": str("ID do projeto para comandos git (obrigatorio com git)")}, "acao"), readonly},
		{"game_booster_estado", "Ler o plano de energia e se ha uma sessao do Game Booster v2.x pendente de restauracao.", schema(map[string]any{}), readonly},
		{"solicitar_fechamento", "SOLICITA fechar apps de uma lista restrita (Chrome, Edge, Discord etc). NAO fecha nada antes de aprovacao no painel do PC. Informe PIDs vistos em listar_processos.", schema(map[string]any{"pids": map[string]any{"type": "array", "items": map[string]any{"type": "integer"}, "minItems": 1, "maxItems": 10}}, "pids"), mutating},
		{"solicitar_plano_energia", "SOLICITA ativar Alto desempenho ou restaurar o plano anterior. NAO altera nada antes de aprovacao no painel do PC.", schema(map[string]any{"acao": map[string]any{"type": "string", "enum": []string{"ativar_alto_desempenho", "restaurar_anterior"}}}, "acao"), mutating},
		{"solicitar_gravacao_arquivo", "SOLICITA gravar texto UTF-8 (ate 32 KiB), somente em pasta com escrita habilitada no painel. Exige aprovacao local; cria backup em substituicoes.", schema(map[string]any{"root_id": str("ID da pasta autorizada para escrita"), "caminho": str("Caminho relativo do arquivo"), "conteudo": str("Conteudo UTF-8 para gravar")}, "root_id", "caminho", "conteudo"), mutating},
		{"agentes_ativos", "Lista agentes reais registrados neste servidor pelo MCP; nenhum dado de demonstracao.", schema(map[string]any{}), readonly},
		{"registrar_agente", "Registrar atividade de um agente/CLI real para visualizacao no Laboratorio de Agentes.", schema(map[string]any{"nome": str("Nome do agente"), "origem": str("Exemplo: codex-cli ou claude-code"), "tarefa": str("Descricao curta")}, "nome", "origem", "tarefa"), mutating},
		{"atualizar_agente", "Enviar status e linha real de saida do agente ao viewer. Status running,waiting,done,error.", schema(map[string]any{"id": str("ID recebido em registrar_agente"), "status": str("running, waiting, done ou error"), "tarefa": str("Atividade atual"), "log": str("Linha de atividade (opcional)")}, "id", "status", "tarefa", "log"), mutating},
		{"listar_arquivos_pc", "Modo aprovacao maxima: lista uma pasta local absoluta (mantem bloqueio para sistema, credenciais, junctions e rede).", schema(map[string]any{"caminho_absoluto": str("Ex: D:\\Projetos")}, "caminho_absoluto"), readonly},
		{"ler_arquivo_pc", "Modo aprovacao maxima: le arquivo UTF-8 ate 64KiB por caminho absoluto, com protecoes de dados sensiveis.", schema(map[string]any{"caminho_absoluto": str("Caminho absoluto do arquivo")}, "caminho_absoluto"), readonly},
		{"gravar_arquivo_pc", "Modo aprovacao maxima: grava UTF-8 ate 32KiB no computador (backup automatico), exceto arquivos protegidos.", schema(map[string]any{"caminho_absoluto": str("Caminho absoluto do arquivo"), "conteudo": str("Texto para gravacao")}, "caminho_absoluto", "conteudo"), mutating},
		{"criar_pasta_pc", "Modo aprovacao maxima: cria uma pasta local com nome seguro; a pasta pai ja deve existir.", schema(map[string]any{"caminho_absoluto": str("Pasta nova, por exemplo D:\\Projetos\\NovaPasta")}, "caminho_absoluto"), mutating},
		{"mover_arquivo_pc", "Modo aprovacao maxima: move ou renomeia um arquivo comum (ate 16 MiB), sem sobrescrever destinos; registra backup recuperavel.", schema(map[string]any{"origem": str("Caminho absoluto do arquivo de origem"), "destino": str("Caminho absoluto para o arquivo de destino")}, "origem", "destino"), mutating},
		{"excluir_arquivo_pc", "Modo aprovacao maxima: remove arquivo comum (ate 16 MiB) da pasta original para a lixeira RECUPERAVEL do Consecom, nunca exclusao permanente.", schema(map[string]any{"caminho_absoluto": str("Arquivo a enviar para lixeira recuperavel")}, "caminho_absoluto"), map[string]any{"readOnlyHint": false, "destructiveHint": true, "openWorldHint": false}},
		{"excluir_pasta_vazia_pc", "Modo aprovacao maxima: remove apenas uma pasta VAZIA; nao apaga pastas com arquivos nem diretórios protegidos.", schema(map[string]any{"caminho_absoluto": str("Pasta vazia a remover")}, "caminho_absoluto"), map[string]any{"readOnlyHint": false, "destructiveHint": true, "openWorldHint": false}},
		{"criar_backup_pc", "Modo aprovacao maxima: arquiva copia recuperavel de arquivo comum (ate 16 MiB) no cofre de recuperacao local protegido pelo perfil do usuario Windows.", schema(map[string]any{"caminho_absoluto": str("Arquivo para copiar para o cofre de recuperacao")}, "caminho_absoluto"), mutating},
		{"listar_recuperacao_pc", "Lista backups e arquivos excluidos recuperaveis no cofre local do Consecom, com IDs para restauracao.", schema(map[string]any{}), readonly},
		{"restaurar_recuperacao_pc", "Modo aprovacao maxima: restaura backup ou arquivo da lixeira por ID; protege o conteudo atual com backup antes de substituir.", schema(map[string]any{"id": str("ID retornado em listar_recuperacao_pc")}, "id"), mutating},
		{"browser_opcoes", "Listar Chrome e Edge instalados, perfis disponiveis e preferencia salva. Se nao houver padrao, pergunte ao usuario qual navegador e perfil; nao escolha sozinho. Perfis personal:* usam sessao isolada. Opcao chrome live:current usa a sessao JA ABERTA autenticada, apos ativar chrome://inspect/#remote-debugging e autorizar no proprio Chrome (144+).", schema(map[string]any{}), readonly},
		{"browser_abrir", "Abrir uma URL HTTP(S) no Chrome/Edge e perfil escolhido. Se um padrao completo estiver salvo no painel local, usa-o; senao DEVE perguntar ao usuario e informar navegador e perfil de browser_opcoes. Perfil personal:* e isolado. Perfil live:current conecta Chrome aberto (com logins atuais), requer depuracao habilitada e permissao local. Se escolher este modo sem acesso local, explique que e necessario consentimento no PC.", schema(map[string]any{"url": str("URL HTTP ou HTTPS"), "navegador": str("chrome ou edge; opcional somente se houver padrao"), "perfil": str("ID exato de browser_opcoes (managed:chatgpt ou personal:...); opcional somente se houver padrao")}, "url"), mutating},
		{"browser_navegar", "Navegar a aba controlada para URL HTTP(S) especifica. Se nao tiver navegador aberto, use browser_abrir.", schema(map[string]any{"url": str("URL completa HTTP ou HTTPS")}, "url"), mutating},
		{"browser_ler_pagina", "Ler URL, titulo, texto visivel (limitado) e elementos interativos da aba controlada, sem acessar senhas de formularios.", schema(map[string]any{}), readonly},
		{"browser_clicar", "Clicar elemento por seletor CSS na aba. Formulario submit, comprar, enviar, excluir, publicar e outras acoes sensiveis estao BLOQUEADAS nesta versao.", schema(map[string]any{"seletor": str("Seletor CSS, por exemplo #pesquisar")}, "seletor"), mutating},
		{"browser_digitar", "Preencher input, textarea ou contenteditable por seletor CSS. Bloqueia senhas e cartoes; nao envia o formulario.", schema(map[string]any{"seletor": str("Seletor CSS do campo"), "texto": str("Texto a digitar, ate 4096 caracteres")}, "seletor", "texto"), mutating},
		{"browser_rolar", "Rolar a aba controlada verticalmente ate 2500 pixels por chamada.", schema(map[string]any{"pixels": map[string]any{"type": "integer", "minimum": -2500, "maximum": 2500}}), mutating},
		{"browser_capturar_tela", "Captura uma imagem JPEG da aba controlada e envia ao ChatGPT. Pode conter dados privados visiveis na tela; use somente mediante pedido de visualizacao.", schema(map[string]any{}), readonly},
		{"browser_fechar_aba", "Fechar apenas a aba criada/controlada pelo Consecom, sem fechar seu navegador pessoal.", schema(map[string]any{}), mutating},
		{"dev_execucao", "Executar comandos FIXOS de diagnóstico de desenvolvimento (git status, diff stat, log, Node/npm version) numa pasta autorizada. npm test/build/lint exigem SEMPRE aprovacao local, pois scripts podem executar codigo arbitrario.", schema(map[string]any{"root_id": str("ID da pasta de projeto autorizada"), "acao": map[string]any{"type": "string", "enum": []string{"git_status", "git_branch", "git_diff_stat", "git_log", "node_version", "npm_version", "npm_test", "npm_build", "npm_lint"}}}, "root_id", "acao"), mutating},
		{"browser_abas", "Lista SOMENTE abas criadas pelo Consecom na sessao controlada; nunca assume abas pessoais existentes.", schema(map[string]any{}), readonly},
		{"browser_nova_aba", "Cria aba nova na sessao atual do Consecom e navega ate a URL.", schema(map[string]any{"url": str("URL HTTPS ou HTTP")}, "url"), mutating},
		{"browser_ativar_aba", "Alterna somente para aba criada pelo Consecom; id retornado em browser_abas.", schema(map[string]any{"id": str("ID de aba pertencente ao Consecom")}, "id"), mutating},
		{"atualizacao_status", "Mostra a versão atual e a política de atualizações remotas definida somente no painel local.", schema(map[string]any{}), readonly},
		{"atualizacao_verificar", "Consulta releases do repositório GitHub confiável cadastrado localmente; não instala nada.", schema(map[string]any{}), readonly},
		{"atualizacao_solicitar", "SOLICITA INSTALAR atualização de um release confiável do GitHub. Verifica SHA-256; exige confirmação no painel, exceto se o usuário habilitou atualizações remotas desse repositório. Reinicia Consecom e pode interromper o túnel temporariamente.", schema(map[string]any{}), map[string]any{"readOnlyHint": false, "destructiveHint": true, "openWorldHint": true}},

		{"dev_job_iniciar","Iniciar comando de desenvolvimento permitido, com limite de tempo e log; scripts/builds sem confirmacao adicional no modo maximo.",schema(map[string]any{"root_id":str("Pasta de projeto autorizada"),"acao":map[string]any{"type":"string","enum":[]string{"git_status","git_diff","git_log","go_test","go_build","npm_test","npm_build","npm_lint"}}},"root_id","acao"),mutating},
		{"dev_job_status","Consultar status e log parcial de uma execucao por ID.",schema(map[string]any{"id":str("ID de execucao")},"id"),readonly},
		{"dev_job_cancelar","Solicitar cancelamento de uma execucao em andamento.",schema(map[string]any{"id":str("ID de execucao")},"id"),mutating},
		{"git_patch_validar","Verificar patch de texto em projeto autorizado sem modificar arquivos.",schema(map[string]any{"root_id":str("ID do projeto"),"patch":str("Diff unificado no formato Git, max 30KB")},"root_id","patch"),readonly},
		{"git_patch_aplicar","Aplicar patch de texto com testes e rollback; no modo maximo executa sem aprovacoes repetidas.",schema(map[string]any{"root_id":str("ID do projeto"),"patch":str("Diff Git"),"teste":map[string]any{"type":"string","enum":[]string{"none","go_test","npm_test","npm_lint"}}},"root_id","patch"),mutating},
		{"git_patch_reverter","Reverter patch registrado pelo Consecom; execucao direta no modo maximo.",schema(map[string]any{"id":str("ID do patch salvo")},"id"),mutating},
		{"git_branch_criar","Criar branch Git em projeto autorizado; execucao direta no modo maximo.",schema(map[string]any{"root_id":str("ID do projeto"),"nome":str("Nome da branch")},"root_id","nome"),mutating},
		{"desktop_janelas","Listar somente janelas visiveis de aplicativos nao protegidos, sem interagir.",schema(map[string]any{}),readonly},
		{"desktop_abrir","Abrir app permitido; modo maximo executa diretamente.",schema(map[string]any{"app":map[string]any{"type":"string","enum":[]string{"notepad","calculator","explorer","vscode"}}},"app"),mutating},
		{"desktop_ativar","Ativar janela visivel; modo maximo executa diretamente.",schema(map[string]any{"window":str("ID numerico da janela como string")},"window"),mutating},
		{"desktop_clicar","Clicar dentro da janela; modo maximo executa diretamente.",schema(map[string]any{"window":str("ID da janela"),"x":map[string]any{"type":"integer"},"y":map[string]any{"type":"integer"}},"window","x","y"),mutating},
		{"desktop_digitar","Digitar texto em janela visivel permitida; modo maximo executa diretamente.",schema(map[string]any{"window":str("ID da janela"),"texto":str("Texto de ate 1000 caracteres")},"window","texto"),mutating},
		{"desktop_capturar","Capturar janela selecionada; modo maximo executa diretamente.",schema(map[string]any{"window":str("ID da janela")},"window"),mutating},
		{"desktop_ultima_captura","Ler captura de janela explicitamente aprovada nos ultimos cinco minutos.",schema(map[string]any{}),readonly},
		{"browser_habilitar_downloads","Permitir downloads apenas na pasta privada do Consecom; exige navegador controlado.",schema(map[string]any{}),mutating},
		{"browser_downloads","Listar downloads concluídos da pasta privada sem ler seus conteudos.",schema(map[string]any{}),readonly},
		{"browser_upload_arquivo","Selecionar arquivo autorizado em campo de upload; modo maximo nao exige confirmacao extra. Nao envia formulario.",schema(map[string]any{"root_id":str("ID da pasta"),"caminho":str("Arquivo relativo"),"seletor":str("Seletor CSS do input[type=file]")},"root_id","caminho","seletor"),mutating},
		{"browser_confirmar_acao","Executar clique para enviar/publicar/excluir; em modo maximo sem confirmacao extra, mantendo validacao de URL/elemento.",schema(map[string]any{"seletor":str("Seletor CSS do botao")},"seletor"),mutating},
		{"browser_download_copiar","Copiar download privado a projeto autorizado sem sobrescrever; modo maximo executa diretamente.",schema(map[string]any{"nome":str("Nome do download"),"root_id":str("ID da pasta"),"destino":str("Destino relativo")},"nome","root_id","destino"),mutating},
		{"codigo_buscar","Pesquisar texto em ate 250 arquivos de codigo de um projeto autorizado, retornando trecho e numero da linha.",schema(map[string]any{"root_id":str("ID do projeto"),"termo":str("Termo de busca")}, "root_id","termo"),readonly},
		{"codigo_ler_trecho","Ler ate 160 linhas de codigo com numero de linha e SHA256 para edicao segura.",schema(map[string]any{"root_id":str("ID do projeto"),"caminho":str("Arquivo relativo"),"inicio":map[string]any{"type":"integer","minimum":1},"quantidade":map[string]any{"type":"integer","minimum":1,"maximum":160}},"root_id","caminho","inicio","quantidade"),readonly},
		{"codigo_substituir_trecho","Substituir trecho unico no codigo com SHA256 de versao e backup, dentro de projeto com escrita permitida.",schema(map[string]any{"root_id":str("ID do projeto"),"caminho":str("Arquivo relativo"),"sha256":str("SHA256 retornado por codigo_ler_trecho"),"original":str("Texto exato a substituir"),"novo":str("Texto substituto")}, "root_id","caminho","sha256","original","novo"),mutating},
		{"codigo_previsualizar_diff","Previsualizar diferencas antes de editar codigo; confirma SHA256 e unicidade sem gravar no disco.",schema(map[string]any{"root_id":str("ID do projeto"),"caminho":str("Arquivo relativo"),"sha256":str("SHA256 da leitura"),"original":str("Trecho original exato"),"novo":str("Trecho proposto")}, "root_id","caminho","sha256","original","novo"),readonly},
		{"codigo_restaurar_backup","Restaurar um backup de codigo pelo ID registrado na edicao; exige checksum do arquivo atual e copia preventiva.",schema(map[string]any{"id":str("ID do backup retornado na edicao")},"id"),mutating},
		{"desktop_controles","Encontrar controles Win32 visiveis associados a uma janela, como botoes, menus e campos.",schema(map[string]any{"window":str("ID da janela")},"window"),readonly},
		{"desktop_controle_clicar","Clicar no centro de um controle Win32 valido, na propria janela. Respeita modo de aprovacao local.",schema(map[string]any{"window":str("ID da janela"),"controle":str("ID do controle obtido em desktop_controles")},"window","controle"),mutating},
		{"programas_criativos","Detectar disponibilidade de FFmpeg, FFprobe, Blender, ImageMagick, Python e Node sem executar tarefas.",schema(map[string]any{}),readonly},
		{"video_inspecionar","Ler metadados de video local do projeto com FFprobe, sem alterar o arquivo.",schema(map[string]any{"root_id":str("ID do projeto"),"caminho":str("Video relativo .mp4 .mov .mkv .webm")},"root_id","caminho"),readonly},
		{"video_extrair_frame","Extrair frame JPEG reduzido (ate 512 KiB) com FFmpeg em timestamp especifico, sem gravar arquivos no projeto.",schema(map[string]any{"root_id":str("ID do projeto"),"caminho":str("Video relativo"),"segundos":map[string]any{"type":"number","minimum":0,"maximum":36000}},"root_id","caminho","segundos"),readonly},
		{"video_diagnosticar","Analisar ate 20 segundos iniciais de video com FFmpeg procurando quadros pretos e congelamentos candidatos.",schema(map[string]any{"root_id":str("ID do projeto"),"caminho":str("Video relativo")},"root_id","caminho"),readonly},
		{"video_render_iniciar","Iniciar exportacao MP4 assincrona em projeto com escrita autorizada (story, landscape, square; ate 180 segundos).",schema(map[string]any{"root_id":str("ID do projeto"),"entrada":str("Video local relativo"),"preset":map[string]any{"type":"string","enum":[]string{"story","landscape","square"}},"inicio":map[string]any{"type":"integer","minimum":0,"maximum":36000},"duracao":map[string]any{"type":"integer","minimum":1,"maximum":180}},"root_id","entrada","preset","inicio","duracao"),mutating},
		{"video_render_status","Consultar status, log, nome da saida e tamanho de exportacao MP4 iniciada pelo Consecom.",schema(map[string]any{"id":str("ID do render")},"id"),readonly},
		{"video_render_cancelar","Interromper exportacao MP4 em andamento e remover arquivo temporario.",schema(map[string]any{"id":str("ID do render")},"id"),mutating},
		{"fluxo_iniciar","Executar fluxo de 1 a 6 comandos de desenvolvimento permitidos de forma sequencial, verificando cada saida, com cancelamento.",schema(map[string]any{"root_id":str("ID do projeto"),"etapas":map[string]any{"type":"array","minItems":1,"maxItems":6,"items":map[string]any{"type":"string","enum":[]string{"git_status","git_diff","git_log","go_test","go_build","npm_test","npm_build","npm_lint"}}}},"root_id","etapas"),mutating},
		{"fluxo_status","Verificar etapa, logs recentes e sucesso/falha de um fluxo em execucao.",schema(map[string]any{"id":str("ID do fluxo")},"id"),readonly},
		{"fluxo_cancelar","Interromper fluxo e sua etapa atualmente em execucao.",schema(map[string]any{"id":str("ID do fluxo")},"id"),mutating},
		{"fluxo_historico","Listar ate 20 fluxos recentes, inclusive execucoes anteriores a reinicializacao do Consecom, com estado real.",schema(map[string]any{}),readonly},
		{"aprovacoes_pendentes", "Mostrar pedidos aguardando aprovacao no painel do proprio PC; o chat nao pode aprova-los.", schema(map[string]any{}), readonly},
	}
}
func argStr(m map[string]json.RawMessage, key string) (string, error) {
	raw, ok := m[key]
	if !ok {
		return "", fmt.Errorf("campo %s obrigatorio", key)
	}
	var s string
	if e := json.Unmarshal(raw, &s); e != nil {
		return "", e
	}
	return s, nil
}
func (a *App) dispatch(name string, raw json.RawMessage) (any, error) {
	args := map[string]json.RawMessage{}
	if len(raw) > 0 && string(raw) != "null" {
		if e := json.Unmarshal(raw, &args); e != nil {
			return nil, e
		}
	}
	switch name {
	case "codigo_previsualizar_diff":
		req:=EditorEdit{}
		var e error
		req.RootID,e=argStr(args,"root_id");if e!=nil{return nil,e}
		req.Path,e=argStr(args,"caminho");if e!=nil{return nil,e}
		req.ExpectedSHA,e=argStr(args,"sha256");if e!=nil{return nil,e}
		req.Old,e=argStr(args,"original");if e!=nil{return nil,e}
		req.New,e=argStr(args,"novo");if e!=nil{return nil,e}
		return a.codeDiffPreview(req)
	case "codigo_restaurar_backup":
		id,e:=argStr(args,"id");if e!=nil{return nil,e}
		return a.codeRestoreRequest(id)
	case "codigo_buscar":
		id,e:=argStr(args,"root_id");if e!=nil{return nil,e}
		term,e:=argStr(args,"termo");if e!=nil{return nil,e}
		return a.codeSearch(id,term)
	case "codigo_ler_trecho":
		id,e:=argStr(args,"root_id");if e!=nil{return nil,e}
		path,e:=argStr(args,"caminho");if e!=nil{return nil,e}
		var start,count int
		if e=json.Unmarshal(args["inicio"],&start);e!=nil{return nil,e}
		if e=json.Unmarshal(args["quantidade"],&count);e!=nil{return nil,e}
		return a.codeSnippet(id,path,start,count)
	case "codigo_substituir_trecho":
		req:=EditorEdit{}
		var e error
		req.RootID,e=argStr(args,"root_id");if e!=nil{return nil,e}
		req.Path,e=argStr(args,"caminho");if e!=nil{return nil,e}
		req.ExpectedSHA,e=argStr(args,"sha256");if e!=nil{return nil,e}
		req.Old,e=argStr(args,"original");if e!=nil{return nil,e}
		req.New,e=argStr(args,"novo");if e!=nil{return nil,e}
		return a.codeEditRequest(req)
	case "desktop_controles":
		id,e:=argStr(args,"window");if e!=nil{return nil,e}
		num,e:=strconv.ParseUint(id,10,64);if e!=nil{return nil,e}
		return desktopChildControls(num)
	case "desktop_controle_clicar":
		id,e:=argStr(args,"window");if e!=nil{return nil,e}
		child,e:=argStr(args,"controle");if e!=nil{return nil,e}
		return a.desktopElementClick(id,child)
	case "programas_criativos":
		return creativeTools()
	case "video_inspecionar":
		id,e:=argStr(args,"root_id");if e!=nil{return nil,e}
		path,e:=argStr(args,"caminho");if e!=nil{return nil,e}
		return a.inspectVideo(id,path)
	case "video_extrair_frame":
		id,e:=argStr(args,"root_id");if e!=nil{return nil,e}
		path,e:=argStr(args,"caminho");if e!=nil{return nil,e}
		var secs float64
		if e=json.Unmarshal(args["segundos"],&secs);e!=nil{return nil,e}
		return a.videoFrame(id,path,secs)
	case "video_diagnosticar":
		id,e:=argStr(args,"root_id");if e!=nil{return nil,e}
		path,e:=argStr(args,"caminho");if e!=nil{return nil,e}
		return a.videoDiagnose(id,path)
	case "video_render_iniciar":
		var spec RenderSpec
		var e error
		spec.RootID,e=argStr(args,"root_id");if e!=nil{return nil,e}
		spec.Input,e=argStr(args,"entrada");if e!=nil{return nil,e}
		spec.Preset,e=argStr(args,"preset");if e!=nil{return nil,e}
		if e=json.Unmarshal(args["inicio"],&spec.Start);e!=nil{return nil,e}
		if e=json.Unmarshal(args["duracao"],&spec.Duration);e!=nil{return nil,e}
		return a.videoRenderRequest(spec)
	case "video_render_status":
		id,e:=argStr(args,"id");if e!=nil{return nil,e}
		return a.videoRenderStatus(id)
	case "video_render_cancelar":
		id,e:=argStr(args,"id");if e!=nil{return nil,e}
		return a.videoRenderCancel(id)
	case "fluxo_iniciar":
		id,e:=argStr(args,"root_id");if e!=nil{return nil,e}
		var steps []string
		if e=json.Unmarshal(args["etapas"],&steps);e!=nil{return nil,e}
		return a.workflowRequest(id,steps)
	case "fluxo_status":
		id,e:=argStr(args,"id");if e!=nil{return nil,e}
		return a.workflowStatus(id)
	case "fluxo_cancelar":
		id,e:=argStr(args,"id");if e!=nil{return nil,e}
		return a.workflowCancel(id)
	case "fluxo_historico":
		return a.workflowListHistory()
	case "dev_job_iniciar":
		id,e:=argStr(args,"root_id");if e!=nil{return nil,e}
		action,e:=argStr(args,"acao");if e!=nil{return nil,e}
		return a.devJobStart(id,action,false)
	case "dev_job_status":
		id,e:=argStr(args,"id");if e!=nil{return nil,e}
		return a.devJobStatus(id)
	case "dev_job_cancelar":
		id,e:=argStr(args,"id");if e!=nil{return nil,e}
		return a.devJobCancel(id)
	case "git_patch_validar","git_patch_aplicar":
		id,e:=argStr(args,"root_id");if e!=nil{return nil,e}
		patch,e:=argStr(args,"patch");if e!=nil{return nil,e}
		if name=="git_patch_validar"{return a.gitPatchCheck(id,patch)}
		return a.gitPatchRequest(id,patch,optionalString(args,"teste"))
	case "git_patch_reverter":
		id,e:=argStr(args,"id");if e!=nil{return nil,e}
		return a.gitPatchRollbackRequest(id)
	case "git_branch_criar":
		id,e:=argStr(args,"root_id");if e!=nil{return nil,e}
		n,e:=argStr(args,"nome");if e!=nil{return nil,e}
		return a.gitBranchRequest(id,n)
	case "desktop_janelas":
		return desktopEntries()
	case "desktop_ultima_captura":
		return desktopLatestCapture()
	case "desktop_abrir","desktop_ativar","desktop_clicar","desktop_digitar","desktop_capturar":
		item,e:=desktopArg(args);if e!=nil{return nil,e}
		switch name{
		case "desktop_abrir": return a.desktopRequest("open",item)
		case "desktop_ativar": return a.desktopRequest("activate",item)
		case "desktop_clicar": return a.desktopRequest("click",item)
		case "desktop_digitar": return a.desktopRequest("type",item)
		default: return a.desktopRequest("capture",item)
		}
	case "browser_habilitar_downloads":
		return a.browserEnableDownloads()
	case "browser_downloads":
		return a.browserDownloads()
	case "browser_upload_arquivo":
		root,e:=argStr(args,"root_id");if e!=nil{return nil,e}
		path,e:=argStr(args,"caminho");if e!=nil{return nil,e}
		selector,e:=argStr(args,"seletor");if e!=nil{return nil,e}
		return a.browserUploadRequest(root,path,selector)
	case "browser_confirmar_acao":
		selector,e:=argStr(args,"seletor");if e!=nil{return nil,e}
		return a.browserSensitiveRequest(selector)
	case "browser_download_copiar":
		name,e:=argStr(args,"nome");if e!=nil{return nil,e}
		root,e:=argStr(args,"root_id");if e!=nil{return nil,e}
		path,e:=argStr(args,"destino");if e!=nil{return nil,e}
		return a.browserDownloadCopyRequest(name,root,path)
	case "dev_execucao":
		rootID, e := argStr(args, "root_id")
		if e != nil {
			return nil, e
		}
		act, e := argStr(args, "acao")
		if e != nil {
			return nil, e
		}
		return a.developerCommand(rootID, act, false)
	case "atualizacao_status":
		return a.updateStatus(), nil
	case "atualizacao_verificar":
		return a.updateCheck()
	case "atualizacao_solicitar":
		return a.updateRequest()
	case "browser_opcoes", "browser_abrir", "browser_navegar", "browser_ler_pagina", "browser_clicar", "browser_digitar", "browser_rolar", "browser_capturar_tela", "browser_fechar_aba", "browser_abas", "browser_nova_aba", "browser_ativar_aba":
		return a.browserCommand(name, args)
	case "agentes_ativos":
		return a.getAgents(), nil
	case "registrar_agente":
		name, e := argStr(args, "nome")
		if e != nil {
			return nil, e
		}
		source, e := argStr(args, "origem")
		if e != nil {
			return nil, e
		}
		task, e := argStr(args, "tarefa")
		if e != nil {
			return nil, e
		}
		return a.registerAgent(name, source, task)
	case "atualizar_agente":
		id, e := argStr(args, "id")
		if e != nil {
			return nil, e
		}
		status, e := argStr(args, "status")
		if e != nil {
			return nil, e
		}
		task, e := argStr(args, "tarefa")
		if e != nil {
			return nil, e
		}
		line, e := argStr(args, "log")
		if e != nil {
			return nil, e
		}
		return a.updateAgent(id, status, task, line)
	case "listar_arquivos_pc", "ler_arquivo_pc", "gravar_arquivo_pc":
		if a.mode() != "maximum" {
			return nil, errors.New("requer modo aprovacao maxima ativado localmente")
		}
		path, e := argStr(args, "caminho_absoluto")
		if e != nil {
			return nil, e
		}
		forWrite := name == "gravar_arquivo_pc"
		target, e := safeFullPath(path, forWrite)
		if e != nil {
			return nil, e
		}
		if name == "listar_arquivos_pc" {
			return safeList(target)
		}
		if name == "ler_arquivo_pc" {
			return safeRead(target)
		}
		value, e := argStr(args, "conteudo")
		if e != nil {
			return nil, e
		}
		if len(value) > 32768 || !utf8.ValidString(value) || strings.ContainsRune(value, 0) {
			return nil, errors.New("conteudo invalido ou acima de 32 KiB")
		}
		p := &Pending{Kind: "write", Path: target, Content: value}
		return a.writeFullApproved(p)
	case "criar_pasta_pc", "mover_arquivo_pc", "excluir_arquivo_pc", "excluir_pasta_vazia_pc", "criar_backup_pc", "listar_recuperacao_pc", "restaurar_recuperacao_pc":
		return a.fileOperation(name, args)
	case "diagnostico_pc":
		return systemMetrics()
	case "listar_processos":
		return processList()
	case "pastas_autorizadas":
		return a.roots(), nil
	case "aprovacoes_pendentes":
		return a.pendings(), nil
	case "game_booster_estado":
		return a.boosterStatus()
	case "listar_arquivos", "ler_arquivo":
		rootID, e := argStr(args, "root_id")
		if e != nil {
			return nil, e
		}
		rel, e := argStr(args, "caminho")
		if e != nil {
			return nil, e
		}
		root, e := a.getRoot(rootID)
		if e != nil {
			return nil, e
		}
		if strings.TrimSpace(rel) == "." && name == "listar_arquivos" {
			if e := ensureNoLinks(root.Path); e != nil {
				return nil, e
			}
			return safeList(root.Path)
		}
		target, e := safePath(root, rel, false)
		if e != nil {
			return nil, e
		}
		if name == "listar_arquivos" {
			return safeList(target)
		}
		return safeRead(target)
	case "terminal_restrito":
		choice, e := argStr(args, "acao")
		if e != nil {
			return nil, e
		}
		if choice == "windows_version" {
			return restrictedCommand(choice, "")
		}
		if choice != "git_status" && choice != "git_log" && choice != "git_branch" {
			return nil, errors.New("comando nao permitido")
		}
		rootID, e := argStr(args, "root_id")
		if e != nil {
			return nil, e
		}
		r, e := a.getRoot(rootID)
		if e != nil {
			return nil, e
		}
		if e := ensureNoLinks(r.Path); e != nil {
			return nil, e
		}
		return restrictedCommand(choice, r.Path)
	case "solicitar_fechamento":
		var pids []int
		if e := json.Unmarshal(args["pids"], &pids); e != nil || len(pids) == 0 || len(pids) > 10 {
			return nil, errors.New("informe 1 a 10 PIDs validos")
		}
		ps, e := processList()
		if e != nil {
			return nil, e
		}
		index := map[int]string{}
		for _, p := range ps {
			index[p.PID] = p.Name
		}
		targets := []ProcessTarget{}
		seen := map[int]bool{}
		for _, id := range pids {
			if seen[id] {
				continue
			}
			seen[id] = true
			n := index[id]
			if !closable[strings.ToLower(n)] {
				return nil, fmt.Errorf("PID %d nao e aplicativo autorizado ou nao aparece na lista", id)
			}
			t := ProcessTarget{PID: id, Name: n}
			if e = validateProcess(t); e != nil {
				return nil, e
			}
			targets = append(targets, t)
		}
		description := fmt.Sprintf("Fechar normalmente %d processo(s): %v. Dados nao salvos podem ser perdidos.", len(targets), targets)
		if a.canAuto("close") {
			for _, t := range targets {
				if e := closeProcess(t); e != nil {
					return nil, e
				}
			}
			return map[string]any{"status": "comandos_de_fechamento_enviados"}, nil
		}
		id, e := a.queue(&Pending{Kind: "close", Description: description, Processes: targets})
		return map[string]any{"solicitacao": id, "status": "aguardando_aprovacao_local", "orientacao": "Abra o painel MCP Controle PC para aprovar ou negar"}, e
	case "solicitar_plano_energia":
		choice, e := argStr(args, "acao")
		if e != nil {
			return nil, e
		}
		if choice != "ativar_alto_desempenho" && choice != "restaurar_anterior" {
			return nil, errors.New("acao nao permitida")
		}
		if choice == "ativar_alto_desempenho" {
			if s := boosterSession(); s["external_booster_session"] == true {
				return nil, errors.New("Game Booster v2 tem restauracao pendente; restaure nele antes")
			}
			if _, e := highPower(); e != nil {
				return nil, e
			}
			a.mu.Lock()
			prior := a.cfg.PreviousPower
			a.mu.Unlock()
			if prior != "" {
				return nil, errors.New("ja existe plano alterado pelo MCP; restaure primeiro")
			}
		} else {
			a.mu.Lock()
			prior := a.cfg.PreviousPower
			a.mu.Unlock()
			if prior == "" {
				return nil, errors.New("nenhum plano anterior registrado")
			}
		}
		if a.canAuto("power") {
			return a.changePower(choice)
		}
		id, e := a.queue(&Pending{Kind: "power", Action: choice, Description: "Alterar o plano de energia do Windows: " + choice})
		return map[string]any{"solicitacao": id, "status": "aguardando_aprovacao_local"}, e
	case "solicitar_gravacao_arquivo":
		rootID, e := argStr(args, "root_id")
		if e != nil {
			return nil, e
		}
		rel, e := argStr(args, "caminho")
		if e != nil {
			return nil, e
		}
		content, e := argStr(args, "conteudo")
		if e != nil {
			return nil, e
		}
		if len(content) > 32768 || !utf8.ValidString(content) || strings.ContainsRune(content, 0) {
			return nil, errors.New("conteudo precisa ser UTF-8 e ter ate 32 KiB")
		}
		r, e := a.getRoot(rootID)
		if e != nil {
			return nil, e
		}
		path, e := safePath(r, rel, true)
		if e != nil {
			return nil, e
		}
		if fi, e := os.Stat(filepath.Dir(path)); e != nil || !fi.IsDir() {
			return nil, errors.New("pasta destino precisa existir")
		}
		if _, e := os.Lstat(path); e != nil && !os.IsNotExist(e) {
			return nil, e
		}
		if a.canAuto("write") {
			return a.writeApproved(&Pending{Kind: "write", Path: path, Content: content})
		}
		id, e := a.queue(&Pending{Kind: "write", Path: path, Content: content, Description: fmt.Sprintf("Gravar %d bytes em %s; backup se existir", len(content), path)})
		return map[string]any{"solicitacao": id, "status": "aguardando_aprovacao_local", "bytes": len(content)}, e
	default:
		return nil, errors.New("ferramenta desconhecida")
	}
}
func (a *App) boosterStatus() (map[string]any, error) {
	power, e := activePower()
	res := boosterSession()
	res["plano_energia_atual"] = power
	a.mu.Lock()
	res["restauracao_mcp_pendente"] = a.cfg.PreviousPower != ""
	a.mu.Unlock()
	return res, e
}
func (a *App) approve(id string) (string, error) {
	p, e := a.pop(id)
	if e != nil {
		return "", e
	}
	switch p.Kind {
	case "update":
		return a.approveUpdate(p)
	case "dev":
		return a.approveDeveloper(p)
	case "devjob":
		return a.approveDevJob(p)
	case "gitpatch":
		return a.gitPatchApproved(p)
	case "gitrollback":
		return a.gitPatchRollbackApproved(p)
	case "gitbranch":
		return a.gitBranchApproved(p)
	case "desktop":
		return a.approveDesktop(p)
	case "codeedit":
		return a.codeEditApproved(p)
	case "coderestore":
		return a.codeRestoreApproved(p)
	case "videorender":
		return a.videoRenderApproved(p)
	case "winchild":
		return a.approveDesktopElement(p)
	case "workflow":
		return a.workflowApproved(p)
	case "browserupload","browserconfirm":
		return a.approveBrowserTransfer(p)
	case "browsercopy":
		return a.approveBrowserCopy(p)
	case "close":
		errs := []string{}
		for _, t := range p.Processes {
			if e := closeProcess(t); e != nil {
				errs = append(errs, fmt.Sprintf("%s/%d: %v", t.Name, t.PID, e))
			}
		}
		if len(errs) > 0 {
			return "", errors.New(strings.Join(errs, " | "))
		}
		return "Comandos de fechamento enviados. Confirme os processos novamente.", nil
	case "power":
		return a.changePower(p.Action)
	case "write":
		return a.writeApproved(p)
	default:
		return "", errors.New("acao invalida")
	}
}
func (a *App) changePower(action string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if action == "ativar_alto_desempenho" {
		if a.cfg.PreviousPower != "" {
			return "", errors.New("sessao MCP ja existe")
		}
		if boosterSession()["external_booster_session"] == true {
			return "", errors.New("sessao externa do Game Booster v2 pendente")
		}
		old, e := activePower()
		if e != nil {
			return "", e
		}
		high, e := highPower()
		if e != nil {
			return "", e
		}
		if strings.EqualFold(old, high) {
			return "Plano de alto desempenho ja estava ativo; nenhuma alteracao", nil
		}
		a.cfg.PreviousPower = old
		a.cfg.ChangedPower = high
		if e := a.saveLocked(); e != nil {
			return "", e
		}
		if e := activatePower(high); e != nil {
			return "", fmt.Errorf("nao confirmou ativacao (%v); plano original ficou salvo para restauracao", e)
		}
		return "Alto desempenho ativado; plano anterior salvo", nil
	}
	if action == "restaurar_anterior" {
		if a.cfg.PreviousPower == "" {
			return "", errors.New("nao ha restauracao registrada")
		}
		if e := activatePower(a.cfg.PreviousPower); e != nil {
			return "", e
		}
		a.cfg.PreviousPower = ""
		a.cfg.ChangedPower = ""
		if e := a.saveLocked(); e != nil {
			return "", e
		}
		return "Plano anterior restaurado", nil
	}
	return "", errors.New("acao invalida")
}
func (a *App) writeApproved(p *Pending) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	var root Root
	found := false
	for _, r := range a.cfg.Roots {
		rel, e := filepath.Rel(r.Path, p.Path)
		if e == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) && r.Writable {
			root = r
			found = true
			break
		}
	}
	if !found {
		return "", errors.New("pasta nao autorizada para escrita no momento")
	}
	rel, _ := filepath.Rel(root.Path, p.Path)
	target, e := safePath(root, rel, true)
	if e != nil {
		return "", e
	}
	if len(p.Content) > 32768 {
		return "", errors.New("conteudo muito grande")
	}
	existing, e := os.ReadFile(target)
	if e != nil && !os.IsNotExist(e) {
		return "", e
	}
	if e == nil {
		if len(existing) > 1024*1024 {
			return "", errors.New("arquivo existente grande demais para backup")
		}
		backup := target + ".bak-" + time.Now().Format("20060102-150405") + "-" + newID()[:8]
		if e := os.WriteFile(backup, existing, 0600); e != nil {
			return "", fmt.Errorf("backup falhou; nada alterado: %w", e)
		}
	}
	tmp := target + ".mcp-tmp-" + newID()
	if e := os.WriteFile(tmp, []byte(p.Content), 0600); e != nil {
		return "", e
	}
	if e = os.Rename(tmp, target); e != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("substituicao cancelada: %w", e)
	}
	return "Arquivo salvo: " + target, nil
}

func (a *App) writeFullApproved(p *Pending) (string, error) {
	if a.mode() != "maximum" {
		return "", errors.New("aprovacao maxima desativada")
	}
	target, e := safeFullPath(p.Path, true)
	if e != nil {
		return "", e
	}
	if fi, e := os.Stat(filepath.Dir(target)); e != nil || !fi.IsDir() {
		return "", errors.New("pasta destino inexistente")
	}
	// Keep both the original sidecar backup and a vault snapshot with a stable ID.
	if _, e := os.Lstat(target); e == nil {
		if _, e := a.archiveFile(target, "backup"); e != nil {
			return "", fmt.Errorf("backup preventivo falhou, nada alterado: %w", e)
		}
	} else if !errors.Is(e, os.ErrNotExist) {
		return "", e
	}
	return writeContentWithBackup(target, p.Content)
}
func writeContentWithBackup(target, content string) (string, error) {
	if len(content) > 32768 {
		return "", errors.New("limite de 32 KiB")
	}
	existing, e := os.ReadFile(target)
	if e != nil && !os.IsNotExist(e) {
		return "", e
	}
	if e == nil {
		if len(existing) > 1024*1024 {
			return "", errors.New("arquivo existente grande demais")
		}
		if e := os.WriteFile(target+".bak-"+time.Now().Format("20060102-150405")+"-"+newID()[:8], existing, 0600); e != nil {
			return "", e
		}
	}
	tmp := target + ".mcp-tmp-" + newID()
	if e = os.WriteFile(tmp, []byte(content), 0600); e != nil {
		return "", e
	}
	if e = os.Rename(tmp, target); e != nil {
		os.Remove(tmp)
		return "", e
	}
	return "Arquivo gravado com backup: " + target, nil
}
