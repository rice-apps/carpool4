package devdb

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestTestConnectionRejectsUnrestrictedCredentials(t *testing.T) {
	for _, dsn := range []string{"", "postgresql://postgres:private-password@localhost/postgres", "postgresql://carpool_development:private-password@localhost/postgres", "not-a-url"} {
		_, _, err := OpenTestPool(context.Background(), dsn)
		if err == nil {
			t.Fatalf("unsafe configuration accepted")
		}
		if strings.Contains(err.Error(), "private-password") {
			t.Fatal("configuration error exposed a password")
		}
	}
}

func TestTestRolesUseDocumentedFeatureAssignments(t *testing.T) {
	for _, role := range []string{"carpool_profile_test", "carpool_ride_detail_test", "carpool_read_lists_test"} {
		if !isTestRole(role) {
			t.Errorf("documented feature role %q was rejected", role)
		}
	}
	for _, role := range []string{"carpool_development", "carpool_pair_01_test", "carpool_profile_test_extra", "carpool_unknown_test"} {
		if isTestRole(role) {
			t.Errorf("unassigned role %q was accepted", role)
		}
	}
}

func TestPairConnectionDoesNotInheritAdministratorQueryCredentials(t *testing.T) {
	config, err := pgx.ParseConfig("postgresql://postgres:admin-password@localhost/postgres?user=postgres&password=query-admin-password&sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	dsn, err := pairConnectionURL(config, "carpool_profile_test", "pair-password")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.User != "carpool_profile_test" || parsed.Password != "pair-password" {
		t.Fatal("pair configuration inherited administrator credentials")
	}
	if strings.Contains(dsn, "admin-password") {
		t.Fatal("pair configuration exposed administrator credentials")
	}
}
