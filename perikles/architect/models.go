package architect

type CnpRule struct {
	Method string `yaml:"method"`
	Path   string `yaml:"path"`
}

type Role struct {
	Privileges []string `yaml:"privileges"`
}

type CnpElasticMapping struct {
	Indices  []string  `yaml:"indices"`
	Role     Role      `yaml:"role"`
	CnpRules []CnpRule `yaml:"cnp_rules"`
}

type CnpRuleSet struct {
	CnpRules []CnpRule
	RoleName string
}
