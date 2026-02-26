package architect

type CnpRule struct {
	Method string `yaml:"method"`
	Path   string `yaml:"path"`
}

type CnpElasticMapping struct {
	CnpRules []CnpRule `yaml:"cnp_rules"`
}

type CnpRuleSet struct {
	CnpRules []CnpRule
	RoleName string
}
