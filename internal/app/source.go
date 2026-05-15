package app

import (
	"context"
	"fmt"

	"github.com/cairon666/agentsflow/internal/builder"
	templatesource "github.com/cairon666/agentsflow/internal/source"
)

func (a App) resolveTemplateSource(ctx context.Context, source string, prompter builder.Prompter) (string, func(), error) {
	resolver := a.SourceResolver
	if resolver == nil {
		resolver = templatesource.NewResolver()
	}
	return resolver.Resolve(ctx, source, templateChooser{prompter: prompter}, a.Stdout)
}

type templateChooser struct {
	prompter builder.Prompter
}

func (c templateChooser) ChooseTemplate(options []templatesource.TemplateOption) (string, error) {
	chooser, ok := c.prompter.(builder.TemplatePrompter)
	if !ok {
		return "", fmt.Errorf("template selection prompt unavailable")
	}
	builderOptions := make([]builder.TemplateOption, 0, len(options))
	for _, option := range options {
		builderOptions = append(builderOptions, builder.TemplateOption{
			Value: option.Value,
			Label: option.Label,
		})
	}
	return chooser.ChooseTemplate(builderOptions)
}
