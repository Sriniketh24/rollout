package targeting

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/Sriniketh24/rollout/internal/models"
)

type Engine struct {
	regexCache sync.Map
}

func New() *Engine {
	return &Engine{}
}

type Segment struct {
	ID          string          `json:"id"`
	ProjectID   string          `json:"project_id"`
	Key         string          `json:"key"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Rules       []models.Clause `json:"rules"`
	Included    []string        `json:"included"`
	Excluded    []string        `json:"excluded"`
}

func (e *Engine) EvaluateClause(clause models.Clause, ctx models.EvalContext) bool {
	attrVal := getAttribute(clause.Attribute, ctx)
	result := e.evaluate(clause.Operator, attrVal, clause.Values)
	if clause.Negate {
		return !result
	}
	return result
}

func (e *Engine) EvaluateRules(rules []models.TargetingRule, ctx models.EvalContext) (matched *models.TargetingRule, ok bool) {
	for i := range rules {
		allMatch := true
		for _, clause := range rules[i].Clauses {
			if !e.EvaluateClause(clause, ctx) {
				allMatch = false
				break
			}
		}
		if allMatch {
			return &rules[i], true
		}
	}
	return nil, false
}

func (e *Engine) MatchesSegment(segment *Segment, ctx models.EvalContext) bool {
	for _, key := range segment.Excluded {
		if ctx.Key == key {
			return false
		}
	}
	for _, key := range segment.Included {
		if ctx.Key == key {
			return true
		}
	}
	for _, clause := range segment.Rules {
		if !e.EvaluateClause(clause, ctx) {
			return false
		}
	}
	return true
}

func getAttribute(attr string, ctx models.EvalContext) string {
	if attr == "key" {
		return ctx.Key
	}
	if v, ok := ctx.Attributes[attr]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func (e *Engine) evaluate(op models.Operator, attrVal string, values []string) bool {
	switch op {
	case models.OpEquals:
		return len(values) > 0 && attrVal == values[0]
	case models.OpNotEquals:
		return len(values) > 0 && attrVal != values[0]
	case models.OpContains:
		return len(values) > 0 && strings.Contains(attrVal, values[0])
	case models.OpStartsWith:
		return len(values) > 0 && strings.HasPrefix(attrVal, values[0])
	case models.OpEndsWith:
		return len(values) > 0 && strings.HasSuffix(attrVal, values[0])
	case models.OpIn:
		for _, v := range values {
			if attrVal == v {
				return true
			}
		}
		return false
	case models.OpNotIn:
		for _, v := range values {
			if attrVal == v {
				return false
			}
		}
		return true
	case models.OpRegex:
		return e.matchRegex(attrVal, values)
	default:
		return false
	}
}

func (e *Engine) matchRegex(attrVal string, values []string) bool {
	if len(values) == 0 {
		return false
	}
	pattern := values[0]
	if cached, ok := e.regexCache.Load(pattern); ok {
		return cached.(*regexp.Regexp).MatchString(attrVal)
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	e.regexCache.Store(pattern, re)
	return re.MatchString(attrVal)
}
