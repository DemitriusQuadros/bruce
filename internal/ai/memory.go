package ai

import (
	"fmt"
	"strings"
	"time"
)

// ToolExecutionGuidelines contains multi-step execution, scheduling, and artifact generation instructions.
const ToolExecutionGuidelines = `

<tool_execution_guidelines>
1. Multi-Step Execution & Tool Chaining:
   - When a user request requires multiple steps (for example, reading a web page or searching AND creating an HTML document/report), you MUST execute all required tools in sequence across turns of the loop before finishing.
   - Do NOT stop after the first tool (e.g. reading a link) to only output a text summary if the user also asked for an HTML artifact, file, or report.
2. Generating HTML & Artifacts:
   - When the user asks for an HTML document, page, resume, dashboard, or report, you MUST invoke the 'artifact_save' tool with complete HTML content in the 'content' field and an appropriate filename (e.g. 'resume.html', 'report.html').
   - The 'content' field is MANDATORY and must contain the full, standalone HTML document (including <!DOCTYPE html>, <html>, <head>, <style>, and <body>). Do NOT call 'artifact_save' with only a filename or title.
   - Keep the HTML design clean, modern, well-structured, and concise. Avoid needlessly repetitive text to ensure fast generation.
   - Once 'artifact_save' succeeds, present the generated URL to the user in your final text response.
3. Scheduling & Proactive Reminders:
   - When the user asks you to schedule a message, report, reminder, or monitor a condition (e.g. 'me manda um hello world daqui a 2 minutos', 'send me a message in 5 minutes', 'watch for emails'), you MUST invoke the 'proactive_create' tool.
   - Bruce supports two execution modes for scheduled tasks:
     a) Direct Message Delivery ('execution_mode': 'message'):
        Use this when the user wants to receive a specific reminder or message at a future time (e.g. 'me manda um hello world daqui a 2 minutos', 'lembre-me de tomar o remédio às 20h', 'send me a reminder').
        In 'prompt_condition', put the exact message text you want delivered to the user (e.g. 'Hello World!' or 'Lembrete: Tomar o remédio'). Do NOT write instructions to an agent like 'Send the message...'. Write the actual message.
     b) AI Agent Execution ('execution_mode': 'agent'):
        Use this when the user wants Bruce to actively research, monitor, use tools, or compile a dynamic report at that time (e.g. 'todo dia às 9h pesquise as notícias de IA e me envie um resumo').
        In 'prompt_condition', write the prompt instructions for the agent.
   - For relative offsets like 'in 2 minutes' or 'daqui a 5 minutos', pass 'type': 'cron' and 'schedule': '+2m' (or '+5m', '+10m', etc.).
   - For recurring crons at a specific time of day (e.g. 'every day at 9am'), pass 'type': 'cron' and standard 5-token cron (e.g. '0 9 * * *').
   - For ambient condition monitoring, pass 'type': 'watch' and the interval in minutes (e.g. '30', minimum 5).
   - NEVER pretend or claim to have scheduled a task or set a reminder unless you have actually called 'proactive_create' and received a successful response from the tool in this turn.
4. Strict Anti-Hallucination:
   - NEVER pretend or claim to have created, saved, or published a file, artifact, or scheduled task unless you have actually called the corresponding tool ('artifact_save', 'proactive_create') and received a successful result in this session.
   - NEVER generate fake URLs like '/artifacts/...' or 'https://.../artifacts/...' in your text without executing the tool first.
</tool_execution_guidelines>`

// AppendToolGuidelines appends tool execution instructions if not already present.
func AppendToolGuidelines(prompt string) string {
	if strings.Contains(prompt, "<tool_execution_guidelines>") {
		return prompt
	}
	return prompt + ToolExecutionGuidelines
}

// FormatTimeGap formats the elapsed time between last interaction and now.
// Returns an empty string if elapsed time is less than 1 hour.
func FormatTimeGap(lastTime time.Time) string {
	if lastTime.IsZero() {
		return ""
	}
	diff := time.Since(lastTime)
	if diff < time.Hour {
		return ""
	}

	var gapStr string
	if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			gapStr = "1 hour ago"
		} else {
			gapStr = fmt.Sprintf("%d hours ago", hours)
		}
	} else {
		days := int(diff.Hours() / 24)
		if days == 1 {
			gapStr = "yesterday"
		} else {
			gapStr = fmt.Sprintf("%d days ago", days)
		}
	}

	return fmt.Sprintf("The last interaction in this conversation was %s (%s). Keep this time gap in mind if the user refers to past discussions or recent events.",
		gapStr, lastTime.Format("2006-01-02 15:04 MST"),
	)
}

// FormatTemporalContext combines current date/time in timezone with any time gap notice.
func FormatTemporalContext(now time.Time, tz string, timeGapNotice string) string {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.Local
	}
	t := now.In(loc)
	out := fmt.Sprintf("Current Date & Time: %s (%s). Timezone: %s.",
		t.Format("Monday, 2006-01-02 15:04:05 -0700"),
		t.Format("2006-01-02 15:04"),
		loc.String(),
	)
	if timeGapNotice != "" {
		out += "\n" + timeGapNotice
	}
	return out
}

// BuildEffectiveSystemPrompt injects conversation summary and temporal context into the base system prompt.
func BuildEffectiveSystemPrompt(basePrompt, summary, temporalNotice string) string {
	prompt := basePrompt
	if summary != "" {
		prompt += fmt.Sprintf("\n\n<conversation_context>\nBelow is a concise summary of previous conversation history and key facts for this session:\n%s\n</conversation_context>", summary)
	}
	if temporalNotice != "" {
		prompt += fmt.Sprintf("\n\n<temporal_context>\n%s\n</temporal_context>", temporalNotice)
	}
	return prompt
}
