package web

import (
	"tldr/internal/config"
	llmgateway "tldr/internal/llm_gateway"
	"tldr/internal/parser"
	"tldr/internal/summarizer"
)

type Container struct {
	Handler    *Handler
	Summarizer *summarizer.Summarizer
	LLMGateway *llmgateway.LlmGateway
	Parser     *parser.Parser
}

func NewContainer(cfg *config.Config) *Container {
	p := parser.NewParser(cfg)
	llm := llmgateway.New()
	s := summarizer.New(p, llm, cfg)
	h := NewHandler(s, cfg)

	return &Container{
		Handler:    h,
		Summarizer: s,
		LLMGateway: llm,
		Parser:     p,
	}
}
