package store

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// Rows written before results carried a workspace move to the prober's
// personal workspace the next time the store starts.
func TestResultsWithoutWorkspaceMoveToOwnersWorkspace(t *testing.T) {
	cfg := Config{BatchSize: 1, FlushInterval: 5 * time.Millisecond}
	st := openTestStore(t, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user, err := st.CreateUser(ctx, "Legacy Prober", testEmail("legacy-results"), "hash", false)
	if err != nil {
		t.Fatalf("CreateUser returned error: %v", err)
	}
	workspace, err := st.GetWorkspaceContextByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetWorkspaceContextByUserID returned error: %v", err)
	}
	if _, err := st.db.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (workspace_id, user_id, ts, protocol, target, seq, rtt_ms)
		VALUES (NULL, $1, NOW(), 'ping', 'legacy.example', 1, 12)
	`, st.tableRef()), user.ID); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	cfg.DSN = st.cfg.DSN
	cfg.TableName = st.tableName
	reopened, err := OpenConfigured(cfg)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	t.Cleanup(func() { _ = reopened.Close() })

	points, err := reopened.QueryHistory(workspace.ID, "ping", "legacy.example", 0, 10)
	if err != nil {
		t.Fatalf("QueryHistory returned error: %v", err)
	}
	if len(points) != 1 {
		t.Fatalf("expected the legacy row in the owner's workspace, got %d rows", len(points))
	}
}

// Alert rules belong to the workspace: other workspaces can't see or delete
// them, and they outlive the account of the member who wrote them.
func TestAlertRulesBelongToWorkspace(t *testing.T) {
	st := openTestStore(t, Config{BatchSize: 1, FlushInterval: 5 * time.Millisecond})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	owner, err := st.CreateUser(ctx, "Team Owner", testEmail("scope-owner"), "hash", false)
	if err != nil {
		t.Fatalf("CreateUser owner returned error: %v", err)
	}
	team, err := st.GetWorkspaceContextByUserID(ctx, owner.ID)
	if err != nil {
		t.Fatalf("GetWorkspaceContextByUserID returned error: %v", err)
	}

	memberEmail := testEmail("scope-member")
	invitation, err := st.CreateWorkspaceInvitationByUserID(ctx, owner.ID, memberEmail, "member", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("CreateWorkspaceInvitationByUserID returned error: %v", err)
	}
	member, err := st.CreateUser(ctx, "Team Member", memberEmail, "hash", false)
	if err != nil {
		t.Fatalf("CreateUser member returned error: %v", err)
	}
	if _, err := st.AcceptWorkspaceInvitationByUserID(ctx, member.ID, invitation.ID); err != nil {
		t.Fatalf("AcceptWorkspaceInvitationByUserID returned error: %v", err)
	}
	personal, err := st.GetWorkspaceContextByWorkspaceIDAndUserID(ctx, mustOwnedWorkspaceID(t, st, member.ID), member.ID)
	if err != nil {
		t.Fatalf("member personal workspace: %v", err)
	}

	rule, err := st.SaveAlertRule(ctx, AlertRule{
		WorkspaceID:         team.ID,
		UserID:              member.ID,
		Name:                "Shared edge",
		Protocol:            "ping",
		Target:              "edge.example",
		RecipientEmail:      owner.Email,
		LatencyThresholdMs:  float64Ptr(100),
		ConsecutiveBreaches: 1,
		CooldownMinutes:     5,
		Enabled:             true,
	})
	if err != nil {
		t.Fatalf("SaveAlertRule returned error: %v", err)
	}

	if rules, err := st.ListAlertRules(ctx, personal.ID); err != nil || len(rules) != 0 {
		t.Fatalf("expected no rules in the member's personal workspace, got %d (err %v)", len(rules), err)
	}
	if err := st.DeleteAlertRule(ctx, personal.ID, rule.ID); !errors.Is(err, ErrAlertRuleNotFound) {
		t.Fatalf("expected ErrAlertRuleNotFound deleting from another workspace, got %v", err)
	}
	rule.Name = "Hijacked"
	rule.WorkspaceID = personal.ID
	if _, err := st.SaveAlertRule(ctx, rule); !errors.Is(err, ErrAlertRuleNotFound) {
		t.Fatalf("expected ErrAlertRuleNotFound updating from another workspace, got %v", err)
	}

	if err := st.DeleteUserAccount(ctx, member.ID); err != nil {
		t.Fatalf("DeleteUserAccount returned error: %v", err)
	}
	rules, err := st.ListAlertRules(ctx, team.ID)
	if err != nil {
		t.Fatalf("ListAlertRules returned error: %v", err)
	}
	if len(rules) != 1 || rules[0].Name != "Shared edge" || rules[0].UserID != 0 {
		t.Fatalf("expected the team rule to survive with no author, got %#v", rules)
	}
}

func mustOwnedWorkspaceID(t *testing.T, st *Store, userID int64) int64 {
	t.Helper()
	var id int64
	if err := st.db.QueryRow(`SELECT id FROM workspaces WHERE owner_user_id = $1`, userID).Scan(&id); err != nil {
		t.Fatalf("owned workspace for user %d: %v", userID, err)
	}
	return id
}

// Row-level security is the backstop for a query that forgets its workspace
// filter: inside a workspace transaction, other workspaces' rows don't exist.
func TestRowLevelSecurityHidesOtherWorkspaces(t *testing.T) {
	st := openTestStore(t, Config{BatchSize: 1, FlushInterval: 5 * time.Millisecond})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var workspaces [2]int64
	for i := range workspaces {
		user, err := st.CreateUser(ctx, "RLS Tester", testEmail(fmt.Sprintf("rls-%d", i)), "hash", false)
		if err != nil {
			t.Fatalf("CreateUser returned error: %v", err)
		}
		workspaces[i] = mustOwnedWorkspaceID(t, st, user.ID)
		if _, err := st.SaveAlertRule(ctx, AlertRule{
			WorkspaceID:         workspaces[i],
			UserID:              user.ID,
			Name:                "RLS rule",
			Protocol:            "ping",
			Target:              "rls.example",
			RecipientEmail:      user.Email,
			LatencyThresholdMs:  float64Ptr(50),
			ConsecutiveBreaches: 1,
			CooldownMinutes:     5,
			Enabled:             true,
		}); err != nil {
			t.Fatalf("SaveAlertRule returned error: %v", err)
		}
		if _, err := st.db.ExecContext(ctx, fmt.Sprintf(`
			INSERT INTO %s (workspace_id, user_id, ts, protocol, target, seq, rtt_ms)
			VALUES ($1, $2, NOW(), 'ping', 'rls.example', 1, 10)
		`, st.tableRef()), workspaces[i], user.ID); err != nil {
			t.Fatalf("insert result: %v", err)
		}
	}
	mine, theirs := workspaces[0], workspaces[1]

	tx, err := st.beginWorkspaceTx(ctx, mine)
	if err != nil {
		t.Fatalf("beginWorkspaceTx returned error: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	// No workspace filter on purpose: the policy has to do the filtering.
	for _, table := range []string{"alert_rules", st.tableRef()} {
		var other int
		if err := tx.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE workspace_id = $1`, table), theirs).Scan(&other); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if other != 0 {
			t.Fatalf("%s: saw %d row(s) from another workspace", table, other)
		}
		var own int
		if err := tx.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE target = 'rls.example'`, table)).Scan(&own); err != nil {
			t.Fatalf("count own %s: %v", table, err)
		}
		if own != 1 {
			t.Fatalf("%s: expected only this workspace's row, got %d", table, own)
		}
	}

	if res, err := tx.ExecContext(ctx, `UPDATE alert_rules SET name = 'Hijacked' WHERE workspace_id = $1`, theirs); err != nil {
		t.Fatalf("update: %v", err)
	} else if n, _ := res.RowsAffected(); n != 0 {
		t.Fatalf("updated %d rule(s) in another workspace", n)
	}

	_, err = tx.ExecContext(ctx, `SAVEPOINT before_insert`)
	if err != nil {
		t.Fatalf("savepoint: %v", err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO alert_rules (workspace_id, name, protocol, target, recipient_email)
		VALUES ($1, 'Planted', 'ping', 'rls.example', 'x@example.com')
	`, theirs)
	if err == nil {
		t.Fatal("expected the policy to reject a rule written into another workspace")
	}
	if _, err := tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT before_insert`); err != nil {
		t.Fatalf("rollback to savepoint: %v", err)
	}

	// The settings end with the transaction, so the pooled connection is clean.
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	var role, setting string
	if err := st.db.QueryRowContext(ctx, `SELECT current_user, COALESCE(current_setting('app.workspace_id', true), '')`).Scan(&role, &setting); err != nil {
		t.Fatalf("read session state: %v", err)
	}
	if role == tenantRole || setting != "" {
		t.Fatalf("connection kept tenant state after the transaction: role=%q workspace=%q", role, setting)
	}
}
