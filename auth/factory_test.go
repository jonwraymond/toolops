package auth

import (
	"testing"
)

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()

	if reg.authenticators == nil {
		t.Error("authenticators map should be initialized")
	}
	if reg.authorizers == nil {
		t.Error("authorizers map should be initialized")
	}
}

func TestRegistry_RegisterAuthenticator(t *testing.T) {
	reg := NewRegistry()

	factory := func(cfg map[string]any) (Authenticator, error) {
		return &mockAuthenticator{name: "test"}, nil
	}

	t.Run("successful registration", func(t *testing.T) {
		err := reg.RegisterAuthenticator("test", factory)
		if err != nil {
			t.Errorf("RegisterAuthenticator() error = %v", err)
		}
	})

	t.Run("duplicate registration", func(t *testing.T) {
		err := reg.RegisterAuthenticator("test", factory)
		if err == nil {
			t.Error("RegisterAuthenticator() should error on duplicate")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		err := reg.RegisterAuthenticator("", factory)
		if err == nil {
			t.Error("RegisterAuthenticator() should error on empty name")
		}
	})

	t.Run("nil factory", func(t *testing.T) {
		err := reg.RegisterAuthenticator("nil_factory", nil)
		if err == nil {
			t.Error("RegisterAuthenticator() should error on nil factory")
		}
	})
}

func TestRegistry_RegisterAuthorizer(t *testing.T) {
	reg := NewRegistry()

	factory := func(cfg map[string]any) (Authorizer, error) {
		return AllowAllAuthorizer{}, nil
	}

	t.Run("successful registration", func(t *testing.T) {
		err := reg.RegisterAuthorizer("test", factory)
		if err != nil {
			t.Errorf("RegisterAuthorizer() error = %v", err)
		}
	})

	t.Run("duplicate registration", func(t *testing.T) {
		err := reg.RegisterAuthorizer("test", factory)
		if err == nil {
			t.Error("RegisterAuthorizer() should error on duplicate")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		err := reg.RegisterAuthorizer("", factory)
		if err == nil {
			t.Error("RegisterAuthorizer() should error on empty name")
		}
	})

	t.Run("nil factory", func(t *testing.T) {
		err := reg.RegisterAuthorizer("nil_factory", nil)
		if err == nil {
			t.Error("RegisterAuthorizer() should error on nil factory")
		}
	})
}

func TestRegistry_CreateAuthenticator(t *testing.T) {
	reg := NewRegistry()

	factory := func(cfg map[string]any) (Authenticator, error) {
		name := "default"
		if n, ok := cfg["name"].(string); ok {
			name = n
		}
		return &mockAuthenticator{name: name}, nil
	}
	_ = reg.RegisterAuthenticator("test", factory)

	t.Run("create registered", func(t *testing.T) {
		auth, err := reg.CreateAuthenticator("test", map[string]any{"name": "custom"})
		if err != nil {
			t.Fatalf("CreateAuthenticator() error = %v", err)
		}
		if auth.Name() != "custom" {
			t.Errorf("Name() = %v, want custom", auth.Name())
		}
	})

	t.Run("create unregistered", func(t *testing.T) {
		_, err := reg.CreateAuthenticator("nonexistent", nil)
		if err == nil {
			t.Error("CreateAuthenticator() should error on unregistered")
		}
	})
}

func TestRegistry_CreateAuthorizer(t *testing.T) {
	reg := NewRegistry()

	factory := func(cfg map[string]any) (Authorizer, error) {
		return AllowAllAuthorizer{}, nil
	}
	_ = reg.RegisterAuthorizer("test", factory)

	t.Run("create registered", func(t *testing.T) {
		auth, err := reg.CreateAuthorizer("test", nil)
		if err != nil {
			t.Fatalf("CreateAuthorizer() error = %v", err)
		}
		if auth.Name() != "allow_all" {
			t.Errorf("Name() = %v, want allow_all", auth.Name())
		}
	})

	t.Run("create unregistered", func(t *testing.T) {
		_, err := reg.CreateAuthorizer("nonexistent", nil)
		if err == nil {
			t.Error("CreateAuthorizer() should error on unregistered")
		}
	})
}

func TestRegistry_ListAuthenticators(t *testing.T) {
	reg := NewRegistry()

	_ = reg.RegisterAuthenticator("z_auth", func(cfg map[string]any) (Authenticator, error) {
		return &mockAuthenticator{name: "z"}, nil
	})
	_ = reg.RegisterAuthenticator("a_auth", func(cfg map[string]any) (Authenticator, error) {
		return &mockAuthenticator{name: "a"}, nil
	})

	names := reg.ListAuthenticators()

	if len(names) != 2 {
		t.Errorf("len(ListAuthenticators()) = %d, want 2", len(names))
	}

	// Should be sorted
	if names[0] != "a_auth" {
		t.Errorf("names[0] = %v, want a_auth", names[0])
	}
	if names[1] != "z_auth" {
		t.Errorf("names[1] = %v, want z_auth", names[1])
	}
}

func TestRegistry_ListAuthorizers(t *testing.T) {
	reg := NewRegistry()

	_ = reg.RegisterAuthorizer("z_authz", func(cfg map[string]any) (Authorizer, error) {
		return AllowAllAuthorizer{}, nil
	})
	_ = reg.RegisterAuthorizer("a_authz", func(cfg map[string]any) (Authorizer, error) {
		return DenyAllAuthorizer{}, nil
	})

	names := reg.ListAuthorizers()

	if len(names) != 2 {
		t.Errorf("len(ListAuthorizers()) = %d, want 2", len(names))
	}

	// Should be sorted
	if names[0] != "a_authz" {
		t.Errorf("names[0] = %v, want a_authz", names[0])
	}
	if names[1] != "z_authz" {
		t.Errorf("names[1] = %v, want z_authz", names[1])
	}
}

func TestDefaultRegistry_BuiltInFactories(t *testing.T) {
	t.Run("jwt authenticator", func(t *testing.T) {
		auth, err := DefaultRegistry.CreateAuthenticator("jwt", map[string]any{
			"secret": "test-secret-key",
		})
		if err != nil {
			t.Fatalf("CreateAuthenticator(jwt) error = %v", err)
		}
		if auth.Name() != "jwt" {
			t.Errorf("Name() = %v, want jwt", auth.Name())
		}
	})

	t.Run("api_key authenticator", func(t *testing.T) {
		auth, err := DefaultRegistry.CreateAuthenticator("api_key", map[string]any{
			"header_name": "X-API-Key",
		})
		if err != nil {
			t.Fatalf("CreateAuthenticator(api_key) error = %v", err)
		}
		if auth.Name() != "api_key" {
			t.Errorf("Name() = %v, want api_key", auth.Name())
		}
	})

	t.Run("oauth2_introspection authenticator", func(t *testing.T) {
		auth, err := DefaultRegistry.CreateAuthenticator("oauth2_introspection", map[string]any{
			"introspection_endpoint": "https://example.com/introspect",
		})
		if err != nil {
			t.Fatalf("CreateAuthenticator(oauth2_introspection) error = %v", err)
		}
		if auth.Name() != "oauth2_introspection" {
			t.Errorf("Name() = %v, want oauth2_introspection", auth.Name())
		}
	})

	t.Run("simple_rbac authorizer", func(t *testing.T) {
		authz, err := DefaultRegistry.CreateAuthorizer("simple_rbac", map[string]any{
			"default_role": "user",
			"roles": map[string]any{
				"admin": map[string]any{
					"permissions": []any{"*"},
				},
			},
		})
		if err != nil {
			t.Fatalf("CreateAuthorizer(simple_rbac) error = %v", err)
		}
		if authz.Name() != "simple_rbac" {
			t.Errorf("Name() = %v, want simple_rbac", authz.Name())
		}
	})

	t.Run("allow_all authorizer", func(t *testing.T) {
		authz, err := DefaultRegistry.CreateAuthorizer("allow_all", nil)
		if err != nil {
			t.Fatalf("CreateAuthorizer(allow_all) error = %v", err)
		}
		if authz.Name() != "allow_all" {
			t.Errorf("Name() = %v, want allow_all", authz.Name())
		}
	})

	t.Run("deny_all authorizer", func(t *testing.T) {
		authz, err := DefaultRegistry.CreateAuthorizer("deny_all", nil)
		if err != nil {
			t.Fatalf("CreateAuthorizer(deny_all) error = %v", err)
		}
		if authz.Name() != "deny_all" {
			t.Errorf("Name() = %v, want deny_all", authz.Name())
		}
	})
}
