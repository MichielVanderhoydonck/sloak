package templates

// Data input
data: #Input

_ruleGroups: #MWMBRRuleGroups & {"data": data}

// Standard Mimir/Loki ruler format
groups: _ruleGroups.groups
