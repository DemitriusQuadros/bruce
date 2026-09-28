# Painel Web & Referência da API REST

O Bruce fornece uma interface web responsiva de gerenciamento e uma API REST completa para administração, auditoria e interação direta.

---

## 1. Visão Geral do Painel Web

O painel está acessível em `http://localhost:8080` (ou `http://<ip-do-servidor>:9090` quando executado via Docker Compose).

- **Assets Embutidos**: Todo o frontend (HTML, CSS, JS, ícones) está compilado diretamente no binário Go através do pacote `embed`. Nenhum servidor web externo é necessário.
- **JavaScript Puro (Vanilla)**: Leve, sem inchaço de frameworks pesados, carregamento instantâneo.

---

## 2. Abas do Painel

### 1. Chat
Interface de conversação integrada:
- Converse com o Bruce diretamente pelo navegador web.
- Crie múltiplos tópicos de conversação independentes.
- Suporte completo a markdown, listas, tabelas e blocos de código com destaque de sintaxe.

### 2. Sessions (Sessões)
Inspecione todas as conversas ativas em qualquer plataforma:
- Filtre sessões por conector (**Discord**, **WhatsApp**, **Telegram**, **Web**).
- Veja o histórico completo de mensagens com timestamps.
- Verifique se há um resumo de longo prazo ativo para cada sessão.
- Exclua sessões (remove em cascata as mensagens e tarefas agendadas correspondentes).

### 3. Schedules (Agendamentos)
Gerencie tarefas proativas e monitoramentos em segundo plano:
- **Badges Visuais**: Diferenciação clara entre `Cron`, `Watch` e modo direto de `Message`.
- **Contagem Regressiva em Tempo Real**: Exibe o tempo restante até a próxima execução agendada.
- **Disparo Imediato**: Botão **Run Now** para enfileirar uma execução de teste imediatamente.
- **Controles**: Pausar, retomar, editar ou excluir qualquer tarefa.
- **Modal de Edição**: Criação de novas tarefas com seleção intuitiva de fuso horário e modo de execução.

### 4. Artifacts (Artefatos)
Galeria de todos os documentos HTML independentes criados pelo Bruce:
- Galeria de cards com títulos, datas e tamanhos de arquivos.
- **Pré-visualização Interativa**: Interaja com o artefato diretamente em um iframe seguro.
- **Alternador de Responsividade**: Teste como o artefato se adapta em Desktop, Tablet e Celular.
- Copie o link compartilhável ou faça download do arquivo HTML.

### 5. Logs
Log de auditoria em tempo real das operações do sistema:
- Filtre por execuções de ferramentas, despachos de mensagens ou erros.
- Inspecione os parâmetros JSON de entrada e saída de cada execução.

### 6. Settings (Configurações)
Editor dinâmico de configurações em tempo de execução:
- Modifique o `ui.default_system_prompt` em tempo real sem reiniciar.
- Alterne o `llm.provider` entre `claude`, `gemini` e `openai`.
- Atualize chaves de API e tokens com campos seguros de senha.
- Ajuste o fuso horário da aplicação (`app.timezone`).

---

## 3. Monitor de Filas Asynq (`/monitor`)

O Bruce embute o [Asynqmon](https://github.com/hibiken/asynqmon) diretamente em `http://localhost:8080/monitor`:
- Visibilidade em tempo real das filas de tarefas do Redis (`default`, `low`, etc.).
- Inspeção de tarefas ativas, pendentes, agendadas, em retentativa e arquivadas.
- Cancelamento manual, exclusão ou reprocessamento de tarefas com falha.

---

## 4. Referência dos Endpoints da API REST

| Método | Endpoint | Descrição |
|---|---|---|
| `GET` | `/health` | Status de integridade e tempo de atividade em segundos. |
| `GET` | `/api/v1/sessions` | Lista todas as sessões com metadados de canal. |
| `GET` | `/api/v1/sessions/{id}` | Obtém uma sessão específica por ID. |
| `DELETE` | `/api/v1/sessions/{id}` | Exclui uma sessão e suas mensagens associadas. |
| `GET` | `/api/v1/chat/sessions` | Lista sessões criadas exclusivamente para o Web Chat. |
| `POST` | `/api/v1/chat/sessions` | Cria uma nova sessão no Web Chat. |
| `POST` | `/api/v1/chat` | Envia mensagem para o Bruce via API do Web Chat. |
| `GET` | `/api/v1/proactive-tasks` | Lista todas as tarefas proativas e watches. |
| `POST` | `/api/v1/proactive-tasks` | Cria nova tarefa proativa (cron ou watch). |
| `GET` | `/api/v1/proactive-tasks/{id}` | Obtém detalhes de uma tarefa agendada. |
| `PATCH` | `/api/v1/proactive-tasks/{id}` | Atualiza agendamento, título, prompt ou status. |
| `POST` | `/api/v1/proactive-tasks/{id}/run` | Dispara manualmente a execução imediata de uma tarefa. |
| `DELETE` | `/api/v1/proactive-tasks/{id}` | Exclui uma tarefa proativa. |
| `GET` | `/api/v1/artifacts` | Lista todos os artefatos HTML salvos. |
| `GET` | `/artifacts/{id}/{filename}` | Serve o arquivo HTML bruto do artefato. |
| `DELETE` | `/api/v1/artifacts/{id}` | Exclui o artefato do disco e do registro. |
| `GET` | `/api/v1/config` | Obtém configurações não-sensíveis em execução. |
| `PUT` | `/api/v1/config` | Atualiza dinamicamente uma configuração. |
| `GET` | `/api/v1/connectors` | Checa status dos conectores Discord, WhatsApp e Telegram. |
| `GET` | `/swagger/` | Documentação interativa do Swagger / OpenAPI. |
