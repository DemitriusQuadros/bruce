# Referência de Ferramentas Nativas

Este documento fornece uma referência completa para todas as ferramentas disponíveis no loop de agente autônomo do Bruce.

---

## Categorias de Ferramentas

- [1. Essenciais & Produtividade](#1-essenciais--produtividade)
  - [`artifact_save`](#artifact_save)
  - [`web_search`](#web_search)
  - [`http_request`](#http_request)
- [2. Proatividade & Agendamento](#2-proatividade--agendamento)
  - [`proactive_create`](#proactive_create)
  - [`proactive_list`](#proactive_list)
  - [`proactive_toggle`](#proactive_toggle)
  - [`proactive_delete`](#proactive_delete)
- [3. Sistema & Arquivos](#3-sistema--arquivos)
  - [`bash_exec`](#bash_exec)
  - [`file_read`, `file_write`, `file_list`](#ferramentas-de-sistema-de-arquivos)
  - [`git_local`](#git_local)
- [4. Integrações & APIs de Terceiros](#4-integracoes--apis-de-terceiros)
  - [Google Workspace (`gmail`, `calendar`, `docs`)](#ferramentas-do-google-workspace)
  - [`github`](#github)
  - [`notion`](#notion)
  - [`trello`](#trello)
  - [`n8n`](#n8n)

---

## 1. Essenciais & Produtividade

### `artifact_save`
Gera e persiste um documento HTML independente salvo em disco e servido via HTTP.

- **Configuração**: `tools.artifacts.enabled: true`
- **Parâmetros**:
  - `title` (*string, obrigatório*): Título legível para o artefato.
  - `filename` (*string, obrigatório*): Nome do arquivo terminado em `.html` (ex: `dashboard.html`).
  - `content` (*string, obrigatório*): Documento HTML autônomo completo incluindo `<!DOCTYPE html>`, `<html>`, `<head>`, `<style>` e `<body>`.
- **Retorno**: Retorna o caminho de acesso público `/artifacts/<id>/<filename>`.

---

### `web_search`
Consulta a API do Google Custom Search para obter informações em tempo real e fontes da web.

- **Configuração**:
  ```yaml
  tools:
    web_search:
      enabled: true
      google_api_key: "AIzaSy..."
      google_cx: "0123456789:abcdef"
  ```
- **Parâmetros**:
  - `query` (*string, obrigatório*): Termos de pesquisa.
  - `num_results` (*inteiro, opcional*): Quantidade de resultados (1-10, padrão: 5).
- **Retorno**: Lista formatada de títulos, trechos e URLs de origem.

---

### `http_request`
Executa requisições HTTP arbitrárias (GET, POST, PUT, DELETE) para APIs REST ou webhooks externos.

- **Configuração**: Habilitado por padrão (`tools.http_client.enabled: true`).
- **Parâmetros**:
  - `url` (*string, obrigatório*): URL completa de destino HTTP/HTTPS.
  - `method` (*string, obrigatório*): Método HTTP (`GET`, `POST`, `PUT`, `DELETE`).
  - `headers` (*objeto, opcional*): Mapa de chave-valor de cabeçalhos.
  - `body` (*string, opcional*): Conteúdo do corpo da requisição.
- **Retorno**: Código de status HTTP e texto de resposta.

---

## 2. Proatividade & Agendamento

### `proactive_create`
Registra um agendamento em segundo plano ou um watch de monitoramento ambiental.

- **Configuração**: Integrado nativamente (requer o poller do agendador).
- **Parâmetros**:
  - `title` (*string, obrigatório*): Nome descritivo da tarefa ou watch.
  - `type` (*string, obrigatório*): `'cron'` para agendamentos baseados em tempo ou `'watch'` para monitoramento periódico de condições.
  - `schedule` (*string, obrigatório*):
    - Para `cron`: Cron padrão de 5 campos (`0 9 * * 1-5`) ou offset relativo (`+2m`, `in 5 minutes`, `daqui a 2 minutos`).
    - Para `watch`: Intervalo de checagem em minutos (string numérica, ex: `'30'`, mínimo 5).
  - `execution_mode` (*string, opcional*):
    - `'message'`: Notificação/lembrete direto (ignora o LLM, 0 tokens, 0 latência).
    - `'agent'`: Executa pelo loop do agente com suporte a ferramentas.
  - `prompt_condition` (*string, obrigatório*):
    - Para `execution_mode: 'message'`: O texto exato a ser entregue.
    - Para `execution_mode: 'agent'`: As instruções para o agente compilar.
    - Para `watch`: A condição a ser monitorada (ex: `'e-mails não lidos de clientes'`).
  - `target_connector` (*string, opcional*): `'discord'`, `'whatsapp'`, `'telegram'` ou `'web'`.
  - `target_channel_id` (*string, opcional*): ID do canal de destino ou número de telefone.
  - `target_tools` (*array de strings, opcional*): Ferramentas autorizadas para o monitoramento.
  - `timezone` (*string, opcional*): Fuso horário IANA (padrão: fuso da aplicação/servidor).

---

### `proactive_list`
Lista todas as tarefas proativas, agendamentos e watches da sessão atual ou de todas as sessões.

---

### `proactive_toggle`
Pausa ou retoma uma tarefa proativa em segundo plano.

- **Parâmetros**:
  - `task_id` (*string, obrigatório*): UUID da tarefa.
  - `is_active` (*booleano, obrigatório*): `true` para retomar, `false` para pausar.

---

### `proactive_delete`
Exclui permanentemente uma tarefa proativa.

- **Parâmetros**:
  - `task_id` (*string, obrigatório*): UUID da tarefa.

---

## 3. Sistema & Arquivos

### `bash_exec`
Executa comandos de terminal em um diretório isolado com timeout de segurança.

- **Configuração**:
  ```yaml
  tools:
    bash:
      enabled: true
      working_dir: "/caminho/do/sandbox"
      timeout_seconds: 30
  ```
- **Parâmetros**:
  - `command` (*string, obrigatório*): Comando de shell a executar.
- **Retorno**: Saída combinada de stdout e stderr.

---

### Ferramentas de Sistema de Arquivos
- `file_read(path)`: Lê o conteúdo de um arquivo local.
- `file_write(path, content)`: Escreve conteúdo em um arquivo local.
- `file_list(directory)`: Lista arquivos e subpastas de um diretório.

---

### `git_local`
Executa operações Git seguras em um repositório local no disco:
- `git_status`: Inspeciona o estado do diretório de trabalho.
- `git_log`: Visualiza o histórico de commits recentes.
- `git_diff`: Mostra alterações em stage ou pendentes.
- `git_commit`: Cria um commit com mensagem.

---

## 4. Integrações & APIs de Terceiros

### Ferramentas do Google Workspace
Requer credenciais do cliente OAuth do Google (`google.oauth_client_id` & `google.oauth_client_secret`).
- **`gmail_search`**: Busca na caixa de entrada usando consultas padrão do Gmail (ex: `is:unread from:chefe`).
- **`gmail_read`**: Lê o corpo do e-mail e metadados da conversa.
- **`gmail_send`**: Redige e envia e-mails.
- **`calendar_list_events`**: Consulta próximos eventos em um intervalo de datas.
- **`calendar_create_event`**: Agenda uma reunião no Google Calendar com título, início e fim.
- **`docs_read` / `docs_create`**: Lê ou cria documentos no Google Docs.

---

### `github`
Interage com issues, pull requests e repositórios no GitHub.
- **Configuração**: `tools.github.api_token: "ghp_..."`
- `github_list_issues`: Lista issues com status, rótulos e responsáveis.
- `github_create_issue`: Abre uma nova issue no repositório.
- `github_list_prs`: Inspeciona pull requests abertos ou mesclados.
- `github_get_repo`: Obtém estatísticas e metadados do repositório.

---

### `notion`
Consulta e atualiza páginas e bancos de dados no Notion.
- **Configuração**: `tools.notion.api_key: "secret_..."`
- `notion_search`: Pesquisa páginas e bancos de dados por título.
- `notion_get_page`: Recupera blocos de conteúdo de uma página.
- `notion_create_page`: Cria uma nova página ou entrada em banco de dados.

---

### `trello`
Gerencia quadros Kanban, listas e cartões no Trello.
- **Configuração**: `tools.trello.api_key` & `tools.trello.user_token`.
- `trello_list_boards`: Lista quadros disponíveis.
- `trello_list_cards`: Obtém cartões de uma lista.
- `trello_create_card`: Cria um novo cartão com título, descrição e prazo.

---

### `n8n`
Dispara fluxos de automação e webhooks no n8n.
- **Configuração**: `tools.n8n.base_url` & `tools.n8n.api_key`.
- `n8n_trigger_webhook`: Envia dados para um nó de webhook do n8n.
- `n8n_execute_workflow`: Dispara a execução de um fluxo via API REST do n8n.
