# Modelo de segurança do atualizador Consecom v2.5

## Trust boundary
- Sem origem padrão. `dono/repositorio` é configurado **no painel local** (POST com CSRF de origem restrita).
- A ferramenta MCP não recebe URL de instalador, token de GitHub nem executa comandos de shell.
- Origem: GitHub Releases `https://github.com/<dono>/<repo>/releases/latest/download/consecom-update.json`.
- O manifesto declara `version`, `url` e `sha256` (minúsculo, 64 hex). URL deve pertencer ao **mesmo** repositório, mesma versão (`releases/download/vX.Y.Z/`) e terminar em `.exe`.
- Downloads apenas HTTPS; redirects apenas para os hosts GitHub de distribuição. Tamanho limite 64 MiB e SHA256 verificado antes e imediatamente antes da execução.
- Não permite downgrade/versão igual.
- Por padrão requer aprovação **presencial no painel**; `remote_approved` fica desativado. Usuário pode ativar opt-in explícito no Windows e então ferramentas remotas podem iniciar instalações autenticadas de releases do repositório escolhido.
- Reinicialização do Consecom e do túnel interrompe temporariamente chamadas atuais. Deve-se retentar após reconexão.
- Recuperação: preserva cópia `MCP-PC.exe.previous` e tenta rollback em falha de inicialização. O rollback não cobre todos os cenários possíveis, como perda de energia.

## Risco residual
Um administrador do GitHub confiável com conta comprometida pode publicar um instalador malicioso e modificar hash/manifesto conjuntamente. Uma evolução indicada é assinatura Ed25519 ou Authenticode com chave offline e release signing automático. Não se deve habilitar `remote_approved` sem controlar esse risco.

Este projeto NÃO instala silenciosamente atualizações de origens não configuradas e NÃO publica release por conta própria.