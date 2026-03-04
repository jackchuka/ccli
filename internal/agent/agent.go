package agent

// Agent is the interface for AI coding agent backends.
type Agent interface {
	Name() string
	ListMCPServers() ([]MCPServer, error)
	GetMCPServer(name string) (*MCPServer, error)
	ListSkills() ([]Skill, error)
	GetSkill(name string) (*Skill, error)
	ListProjects() ([]Project, error)
	GetProject(path string) (*Project, error)
	ListRules() ([]Rule, error)
	GetRule(name string) (*Rule, error)
	Info() (*InstallInfo, error)
}
