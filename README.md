# Consecom Plugin

Plugin Windows para integração controlada entre ChatGPT e o computador local, por túnel MCP.

## Estado da distribuição

**Atenção:** este repositório está sendo preparado para receber o código-fonte e a release Windows **v2.6**. Enquanto o código e a release não estiverem efetivamente enviados, não há atualização baixável automaticamente por este repositório.

O atualizador do Consecom consulta releases do repositório configurado no painel, confere versão e SHA-256 do instalador e solicita aprovação local conforme a política selecionada.

## Funcionalidades do projeto v2.6

- Acesso a arquivos e pastas do Windows com proteções de sistema e credenciais
- Controle de Chrome/Edge com perfis isolados e opção experimental de sessão existente no Chrome
- Múltiplas abas controladas
- Terminal de desenvolvimento com ações restritas e aprovação para scripts npm
- Fila persistente de tarefas e painel local
- Atualizações de versões publicadas no GitHub Releases

## Publicação de atualizações

O workflow `.github/workflows/publicar-release.yml` destina-se a compilar o código Go no Windows em uma tag `vX.Y.Z`, validar testes e publicar dois assets:

- `Consecom_Plugin_vX.Y.Z.exe`
- `consecom-update.json` com `version`, `url` e `sha256`

O workflow requer o **código Go completo** antes de ser executado e permissão `contents: write` para o `GITHUB_TOKEN`. A versão declarada no código deve corresponder à tag.

## Segurança

Não publique tokens do túnel, chaves da OpenAI Platform, credenciais, perfis de navegador ou estado local. Nunca anexe dados reais de sessão ao repositório. Scripts npm podem executar código de projetos e exigem aprovação presencial. O acesso remoto ao navegador mantém proteções para ações sensíveis.

## Autonomia da IA

O painel pode receber tarefas e processar diagnósticos locais predefinidos, mas **não faz esta conversa do ChatGPT executar sozinha a cada 10 segundos**. Um executor de IA independente é outra integração e não está incluído na v2.6.
