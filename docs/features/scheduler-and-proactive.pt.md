# Agendador & Tarefas Proativas

O Bruce possui um agendador autônomo em segundo plano que permite tanto **notificações futuras diretas** quanto **briefings agendados com agentes de IA** através de múltiplos canais de mensagens.

---

## 1. Visão Geral da Arquitetura

O agendador é composto por três sistemas integrados:
1. **Armazenamento SQLite (tabela `proactive_tasks`)**: Armazena agendamentos, condições, canais de destino e estados.
2. **Poller em Segundo Plano (`internal/scheduler/poller.go`)**: Executa a cada minuto, localiza tarefas vencidas (`next_run_at <= now AND is_active = 1`), recalcula os próximos horários de execução e enfileira os trabalhos no Redis.
3. **Motor do Worker Asynq (`internal/worker/proactive_handlers.go`)**: Desenfileira e executa as tarefas de acordo com o **Modo de Execução** configurado.

---

## 2. Modos Duplos de Execução

O Bruce distingue explicitamente entre **Entrega Direta de Mensagem** e **Briefing do Agente de IA**:

```
┌────────────────────────────────────────────────────────┐
│               Tarefa Agendada (Cron)                   │
└──────────────────────────┬─────────────────────────────┘
                           │
             ┌─────────────┴─────────────┐
             ▼                           ▼
    execution_mode: "message"   execution_mode: "agent"
     (Notificação Direta)        (Tarefa Dinâmica de IA)
             │                           │
     Ignora o LLM                Executa Loop do Agente
     0 tokens, 0 latência        Executa ferramentas e compila relatório
```

### Modo 1: Entrega Direta de Mensagem (`execution_mode: "message"`)
Projetado para **lembretes, alarmes e notificações agendadas específicas**.
- **Ignora o LLM completamente** no momento do disparo.
- **Zero latência, zero custo de tokens**, totalmente imune a limites de taxa (rate limits) ou esgotamento de cotas de API.
- Extrai e limpa o texto da mensagem diretamente de `prompt_condition` (removendo aspas envolventes ou comandos repetitivos como `"Envie a mensagem: ..."`).
- Envia a mensagem diretamente para o Discord, WhatsApp ou Telegram.

**Exemplos Típicos**:
- *"Me manda um hello world daqui a dois minutos"*
- *"Lembre-me de tomar o remédio de pressão às 20h"*
- *"Mande para mim 'Ligar para o dentista' às 14:30"*

---

### Modo 2: Modo Briefing do Agente (`execution_mode: "agent"`)
Projetado para **pesquisas dinâmicas, verificações ambientais e resumos gerados por ferramentas**.
- No horário agendado, o worker invoca o **loop autônomo do agente** (`RunAgentLoop`).
- O Bruce pode navegar na web, verificar pull requests no GitHub, ler arquivos ou consultar bancos de dados.
- Os resultados são sintetizados em um relatório formatado e enviados para o seu canal.

**Exemplos Típicos**:
- *"Todo dia útil às 9h, pesquise as notícias de IA e me envie um briefing de 3 tópicos."*
- *"Verifique as issues do GitHub toda manhã e me alerte se houver novos bugs críticos."*

---

## 3. Sintaxe de Agendamento Suportada

### A. Deslocamentos Relativos (Tarefas Pontuais)
Você pode agendar tarefas usando expressões de tempo relativo naturais:
- `+2m`, `2m`, `in 2 minutes`, `daqui a 2 minutos`
- `+30m`, `in 1 hour`, `daqui a 3 horas`

**Como funciona nos bastidores**:
1. Quando `proactive_create` recebe um deslocamento relativo (ex: `+2m`), ele calcula o timestamp exato de destino no fuso horário configurado.
2. Ele converte o offset relativo em uma expressão cron de 5 campos com data específica:
   `minuto hora dia mês *` (ex: `15 14 28 9 *`).
3. Quando o poller avalia essa tarefa após a execução, detecta que a execução subsequente está a mais de 30 dias no futuro (ano seguinte) e marca automaticamente `is_active = 0` (desativando-a para que nunca se repita).

### B. Expressões Cron Recorrentes de 5 Campos
Sintaxe cron padrão suportada por `robfig/cron/v3`:
```text
┌───────────── minuto (0 - 59)
│ ┌─────────── hora (0 - 23)
│ │ ┌───────── dia do mês (1 - 31)
│ │ │ ┌─────── mês (1 - 12)
│ │ │ │ ┌───── dia da semana (0 - 6, Domingo a Sábado)
│ │ │ │ │
* * * * *
```
- `0 9 * * 1-5`: Todo dia útil às 09:00.
- `*/15 * * * *`: A cada 15 minutos.
- `0 18 * * 5`: Toda sexta-feira às 18:00.

### C. Monitoramento de Condições Ambientais (`task_type: "watch"`)
Watches executam periodicamente para monitorar condições em vez de entregar um agendamento fixo:
- **Intervalo**: Especificado em minutos inteiros (ex: `30` ou `60`). Impõe um **limite mínimo estrito de 5 minutos** para evitar limites de taxa de API.
- **Condição**: Descrição em linguagem natural do que monitorar (ex: *"e-mails não lidos de prestadores de serviço"*).
- **Direcionamento de Ferramentas**: Especifica `target_tools` (ex: `["gmail_search"]`) para limitar o uso de ferramentas específicas.
- **Hashing de Desduplicação**: O Bruce gera um hash do resultado da avaliação (`last_result_hash`). Se a condição não mudou desde a última checagem, nenhum alerta é disparado, evitando spam de notificações.

---

## 4. Roteamento Entre Canais (Cross-Channel)

Os agendamentos não estão presos ao canal onde foram criados:
- Você pode conversar com o Bruce no **Discord** e agendar um alerta para ser entregue no seu número do **WhatsApp**:
  ```json
  {
    "target_connector": "whatsapp",
    "target_channel_id": "5511999999999"
  }
  ```
- Ou agendar um relatório a partir do **Painel Web** para ser entregue em um canal privado de DM no **Discord**.

---

## 5. Ferramentas de Gerenciamento Conversacional

Você pode gerenciar todas as tarefas conversando através de qualquer conector de chat:

| Ferramenta | Propósito | Exemplo de Prompt Conversacional |
|---|---|---|
| `proactive_create` | Cria um agendamento ou watch | *"Me lembre de comprar café daqui a 10 minutos"* |
| `proactive_list` | Lista todas as tarefas agendadas | *"Quais são os meus agendamentos ativos?"* |
| `proactive_toggle` | Pausa ou retoma uma tarefa | *"Pausa o meu relatório diário de notícias"* |
| `proactive_delete` | Exclui uma tarefa agendada | *"Deleta o lembrete de café"* |

---

## 6. API REST & Painel Web

Você também pode gerenciar tarefas programaticamente:
- `GET /api/v1/proactive-tasks` — Lista todas as tarefas.
- `POST /api/v1/proactive-tasks` — Cria uma nova tarefa proativa.
- `PATCH /api/v1/proactive-tasks/{id}` — Atualiza agendamento, modo de execução ou status ativo.
- `POST /api/v1/proactive-tasks/{id}/run` — Dispara manualmente uma execução imediata para testes.
- `DELETE /api/v1/proactive-tasks/{id}` — Exclui uma tarefa.

A aba **Schedules** no Painel Web (`http://localhost:8080`) fornece controles visuais, contadores regressivos, badges de modo de execução e um modal de edição.
