package paynow

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ProfileConfig struct {
	APIKey         string `json:"api_key"`
	AuthKind       string `json:"auth_kind,omitempty"`
	AuthPrefix     string `json:"auth_prefix,omitempty"`
	StoreID        string `json:"store_id,omitempty"`
	BaseURL        string `json:"base_url,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

type Profile struct {
	Name       string
	Client     *Client
	StoreID    string
	AuthPrefix string
}

type ProfileSummary struct {
	Name       string `json:"name"`
	AuthPrefix string `json:"auth_prefix"`
	StoreID    string `json:"store_id,omitempty"`
	IsDefault  bool   `json:"is_default"`
}

type Registry struct {
	profiles    map[string]Profile
	defaultName string
}

func RegistryFromEnv() (*Registry, error) {
	configs, defaultName, err := profileConfigsFromEnv()
	if err != nil {
		return nil, err
	}
	return NewRegistry(configs, defaultName), nil
}

func NewRegistry(configs map[string]ProfileConfig, defaultName string) *Registry {
	profiles := make(map[string]Profile, len(configs))
	for name, cfg := range configs {
		authPrefix := strings.TrimSpace(cfg.AuthPrefix)
		if authPrefix == "" {
			authPrefix = authPrefixFromKind(cfg.AuthKind)
		}

		timeout := 30 * time.Second
		if cfg.TimeoutSeconds > 0 {
			timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
		}

		client := NewClient(Config{
			APIKey:     cfg.APIKey,
			AuthPrefix: authPrefix,
			BaseURL:    cfg.BaseURL,
			Timeout:    timeout,
			UserAgent:  "paynow-mcp",
		})

		profiles[name] = Profile{
			Name:       name,
			Client:     client,
			StoreID:    strings.TrimSpace(cfg.StoreID),
			AuthPrefix: authPrefix,
		}
	}

	return &Registry{
		profiles:    profiles,
		defaultName: defaultName,
	}
}

func (r *Registry) Get(name string) (Profile, error) {
	if name == "" {
		name = r.defaultName
	}
	profile, ok := r.profiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("unknown PayNow profile %q; known profiles: %s", name, strings.Join(r.Names(), ", "))
	}
	return profile, nil
}

func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.profiles))
	for name := range r.profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r *Registry) Summaries() []ProfileSummary {
	names := r.Names()
	summaries := make([]ProfileSummary, 0, len(names))
	for _, name := range names {
		profile := r.profiles[name]
		summaries = append(summaries, ProfileSummary{
			Name:       name,
			AuthPrefix: profile.AuthPrefix,
			StoreID:    profile.StoreID,
			IsDefault:  name == r.defaultName,
		})
	}
	return summaries
}

func profileConfigsFromEnv() (map[string]ProfileConfig, string, error) {
	profilesJSON := strings.TrimSpace(os.Getenv("PAYNOW_PROFILES"))
	singleAPIKey := strings.TrimSpace(os.Getenv("PAYNOW_API_KEY"))

	configs := map[string]ProfileConfig{}
	if profilesJSON != "" {
		if err := json.Unmarshal([]byte(profilesJSON), &configs); err != nil {
			return nil, "", fmt.Errorf("PAYNOW_PROFILES is not valid JSON: %w", err)
		}
		if len(configs) == 0 {
			return nil, "", errors.New("PAYNOW_PROFILES must contain at least one profile")
		}
	}

	if singleAPIKey != "" {
		configs["default"] = ProfileConfig{
			APIKey:         singleAPIKey,
			AuthKind:       os.Getenv("PAYNOW_AUTH_KIND"),
			AuthPrefix:     os.Getenv("PAYNOW_AUTH_PREFIX"),
			StoreID:        os.Getenv("PAYNOW_STORE_ID"),
			BaseURL:        os.Getenv("PAYNOW_BASE_URL"),
			TimeoutSeconds: timeoutSecondsFromEnv(),
		}
	}

	if len(configs) == 0 {
		return nil, "", errors.New("set PAYNOW_API_KEY or PAYNOW_PROFILES")
	}

	defaultName := strings.TrimSpace(os.Getenv("PAYNOW_DEFAULT_PROFILE"))
	if defaultName == "" {
		if _, ok := configs["default"]; ok {
			defaultName = "default"
		} else {
			names := make([]string, 0, len(configs))
			for name := range configs {
				names = append(names, name)
			}
			sort.Strings(names)
			defaultName = names[0]
		}
	}
	if _, ok := configs[defaultName]; !ok {
		return nil, "", fmt.Errorf("PAYNOW_DEFAULT_PROFILE %q was not found in profiles", defaultName)
	}

	for name, cfg := range configs {
		cfg.APIKey = strings.TrimSpace(cfg.APIKey)
		cfg.AuthKind = strings.TrimSpace(cfg.AuthKind)
		cfg.AuthPrefix = strings.TrimSpace(cfg.AuthPrefix)
		cfg.StoreID = strings.TrimSpace(cfg.StoreID)
		cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
		if cfg.APIKey == "" {
			return nil, "", fmt.Errorf("PayNow profile %q is missing api_key", name)
		}
		configs[name] = cfg
	}

	return configs, defaultName, nil
}

func timeoutSecondsFromEnv() int {
	raw := strings.TrimSpace(os.Getenv("PAYNOW_TIMEOUT_SECONDS"))
	if raw == "" {
		return 0
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0
	}
	return value
}
