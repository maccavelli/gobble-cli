package cli

import (
	"fmt"
	"path/filepath"

	"github.com/maccavelli/gobble-cli/acpserver"
	"github.com/maccavelli/gobble-cli/internal/config"
	"github.com/maccavelli/gobble-cli/permission"
)

// permissionOptions are where the agent saves rules (0005-PLAN F6-2b
// decision E): permissions.json in the config directory, and the global
// settings.yaml, created from the template when it is missing. With no
// config directory the store is none.
func permissionOptions() acpserver.PermissionOptions {
	o := acpserver.PermissionOptions{AddGlobal: addGlobalRule, RemoveGlobal: removeGlobalRule}
	if d, _, err := systemDirs(); err == nil {
		o.Store = permission.RuleStore{Path: filepath.Join(d.Config, permission.PermissionsFile)}
	}
	return o
}

// addGlobalRule appends rule to the global settings file's
// permissions.rules, and gives the file's rules after.
func addGlobalRule(rule permission.Spec) ([]permission.Spec, error) {
	file, layer, err := settingsFile(false)
	if err != nil {
		return nil, err
	}
	if err := prepare(file, layer, false); err != nil {
		return nil, err
	}
	if err := config.AppendRule(file, configRule(rule)); err != nil {
		return nil, err
	}
	return globalRules(file), nil
}

// removeGlobalRule removes rule, at its line, from the global settings
// file, and gives the file's rules after.
func removeGlobalRule(rule permission.Spec) ([]permission.Spec, error) {
	file, _, err := settingsFile(false)
	if err != nil {
		return nil, err
	}
	if rule.Source.File != file {
		return nil, fmt.Errorf("%s is set in %s, not in %s", rule, rule.Source, file)
	}
	if err := config.RemoveRule(file, rule.Source.Line, configRule(rule)); err != nil {
		return nil, err
	}
	return globalRules(file), nil
}

// globalRules are the global settings file's own rules, each with its
// file and line.
func globalRules(file string) []permission.Spec {
	r, _ := config.Resolve(config.Options{ConfigDir: filepath.Dir(file)})
	return rulesOf(r.Settings.Permissions.Rules)
}

// configRule is rule as a settings file writes it.
func configRule(rule permission.Spec) config.PermissionRule {
	switch rule.Effect {
	case permission.ModeAllow:
		return config.PermissionRule{Allow: rule.Rule()}
	case permission.ModeAsk:
		return config.PermissionRule{Ask: rule.Rule()}
	}
	return config.PermissionRule{Deny: rule.Rule()}
}
