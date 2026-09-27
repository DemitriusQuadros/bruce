package ai

import (
	"fmt"
	"strings"
	"time"
)

// ToolExecutionGuidelines contains multi-step execution and artifact generation instructions.
const ToolExecutionGuidelines = `

<tool_execution_guidelines>
1. Multi-Step Execution & Tool Chaining:
   - When a user request requires multiple steps (for example, reading a web page or searching AND creating an HTML document/report), you MUST execute all required tools in sequence across turns of the loop before finishing.
   - Do NOT stop after the first tool (e.g. reading a link) to only output a text summary if the user also asked for an HTML artifact, file, or report.
2. Generating HTML & Artifacts:
   - When the user asks for an HTML document, page, resume, dashboard, or report, you MUST invoke the 'artifact_save' tool with complete HTML content and an appropriate filename (e.g. 'resume.html', 'report.html').
   - Provide clean, modern HTML with inline CSS styling in the 'content' field.
   - Once 'artifact_save' succeeds, present the generated URL to the user in your final text response.
3. Strict Anti-Hallucination:
   - NEVER pretend or claim to have created, saved, or published a file or artifact unless you have actually called 'artifact_save' and received a successful result from the tool in this session.
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

// BuildEffectiveSystemPrompt injects conversation summary and temporal context into the base system prompt.
func BuildEffectiveSystemPrompt(basePrompt, summary, timeGapNotice string) string {
	prompt := basePrompt
	if summary != "" {
		prompt += fmt.Sprintf("\n\n<conversation_context>\nBelow is a concise summary of previous conversation history and key facts for this session:\n%s\n</conversation_context>", summary)
	}
	if timeGapNotice != "" {
		prompt += fmt.Sprintf("\n\n<temporal_context>\n%s\n</temporal_context>", timeGapNotice)
	}
	return prompt
}
