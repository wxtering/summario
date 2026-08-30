package prompt

import (
	"fmt"
	"tldr/internal/models"
)

const (
	baseSystemPrompt = "You are an objective AI summarizer. Your task is to extract the core ideas, key insights, and factual context from the provided text for informational, educational, and news analysis purposes."

	strategyTLDRInstruction         = "Provide a concise TL;DR summary in 2-4 sentences capturing the main point."
	strategyDetailedInstruction     = "Provide a comprehensive, section-by-section structured summary covering all key details."
	strategyBulletPointsInstruction = "Extract 3 to 7 key takeaways formatted as a bulleted list."

	formatMarkdownInstruction  = "Inside the 'summary' JSON field, format the text using Telegram-compatible MarkdownV2 (e.g. *bold*, _italic_, ~strikethrough~, `code`). Do NOT use unsupported Markdown features."
	formatPlainTextInstruction = "Inside the 'summary' JSON field, format the text strictly as clean plain text without Markdown formatting, asterisks, or hashtags."
	formatHTMLInstruction      = "Inside the 'summary' JSON field, format the text using strictly Telegram-compatible HTML tags only: <b>, <strong>, <i>, <em>, <u>, <ins>, <s>, <strike>, <del>, <span class=\"tg-spoiler\">, <a>, <code>, and <pre>. Do NOT use block elements like <p>, <ul>, <li>, <h1>-<h6> or any other unsupported tags."

	jsonOutputInstruction = "You MUST return your response ONLY as a raw, valid JSON object with exactly two string fields: \"title\" (a concise title for the content) and \"summary\" (the generated summary text). Do NOT wrap the JSON response in markdown code blocks."
)

func BuildPrompt(format models.Format, language string, strategy models.Strategy) string {
	return fmt.Sprintf("%s Write the response strictly in %s language. %s %s %s",
		baseSystemPrompt,
		language,
		getStrategyInstruction(strategy),
		getFormatInstruction(format),
		jsonOutputInstruction,
	)
}

func getStrategyInstruction(strategy models.Strategy) string {
	switch strategy {
	case models.StrategyDetailed:
		return strategyDetailedInstruction
	case models.StrategyBulletPoints:
		return strategyBulletPointsInstruction
	case models.StrategyTLDR:
		fallthrough
	default:
		return strategyTLDRInstruction
	}
}

func getFormatInstruction(format models.Format) string {
	switch format {
	case models.FormatPlainText:
		return formatPlainTextInstruction
	case models.FormatHTML:
		return formatHTMLInstruction
	case models.FormatMarkdown:
		fallthrough
	default:
		return formatMarkdownInstruction
	}
}
