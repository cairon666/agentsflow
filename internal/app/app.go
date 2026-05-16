package app

// App owns the CLI use cases.
type App struct {
	TemplateSource TemplateSource
	FlowLoader     FlowLoader
	TargetRegistry TargetRegistry
	InstallPlanner InstallPlanner
	InstallWriter  InstallWriter
	Reporter       Reporter
	WorkDir        string
	HomeDir        string
}
