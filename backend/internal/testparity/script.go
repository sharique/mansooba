package testparity

import "strings"

const (
	adminEmail = "admin@example.com"
	demoEmail  = "demo@example.com"
	thirdEmail = "third@example.com"
	password   = "Passw0rd!x"
)

type m = map[string]any

// Script returns the fixed request sequence. It assumes an empty database
// (first-run state) and a server configured with a body limit of 4M and a
// high enough auth rate limit that the script itself is never limited; the
// limiter's own responses are characterised separately by the httpserver
// behaviour tests, where the limit can be made tiny and deterministic.
//
// Attachment and avatar steps need S3 (LocalStack) and are added by
// StorageSteps when it is available.
func Script() []Step {
	s := []Step{
		// ---- public and router-level behaviour ----
		{Name: "health", Method: "GET", Path: "/health"},
		{Name: "about", Method: "GET", Path: "/api/v1/about"},
		{Name: "setup_status_before", Method: "GET", Path: "/api/v1/setup/status"},
		{Name: "router_not_found", Method: "GET", Path: "/no-such-route"},
		{Name: "router_not_found_in_api", Method: "GET", Path: "/api/v1/no-such-route"},
		{Name: "router_method_not_allowed", Method: "DELETE", Path: "/health"},
		{Name: "cors_preflight_allowed", Method: "OPTIONS", Path: "/api/v1/auth/login",
			Headers: map[string]string{"Origin": "http://localhost:3000", "Access-Control-Request-Method": "POST"}},
		{Name: "cors_preflight_other_origin", Method: "OPTIONS", Path: "/api/v1/auth/login",
			Headers: map[string]string{"Origin": "http://evil.example", "Access-Control-Request-Method": "POST"}},
		{Name: "cors_simple_request_allowed_origin", Method: "GET", Path: "/health",
			Headers: map[string]string{"Origin": "http://localhost:3000"}},
		{Name: "protected_without_token", Method: "GET", Path: "/api/v1/projects"},
		{Name: "protected_bad_token", Method: "GET", Path: "/api/v1/projects", Headers: map[string]string{"Authorization": "Bearer not-a-token"}},
		{Name: "swagger_index", Method: "GET", Path: "/swagger/index.html"},
		{Name: "avatar_missing", Method: "GET", Path: "/avatars/nobody.png"},
		{Name: "body_too_large", Method: "POST", Path: "/api/v1/auth/login", Raw: []byte(`{"email":"` + strings.Repeat("a", 5<<20) + `"}`)},
		{Name: "invalid_json", Method: "POST", Path: "/api/v1/auth/login", Raw: []byte(`{not json`)},

		// ---- first-run setup ----
		{Name: "setup_admin_invalid", Method: "POST", Path: "/api/v1/setup/admin", Body: m{}},
		{Name: "setup_admin_weak_password", Method: "POST", Path: "/api/v1/setup/admin", Body: m{"full_name": "A", "email": adminEmail, "password": "weak"}},
		{Name: "setup_admin", Method: "POST", Path: "/api/v1/setup/admin",
			Body:    m{"full_name": "Admin User", "email": adminEmail, "password": password},
			Capture: m2("admin_token", "access_token", "admin_id", "user.id")},
		{Name: "setup_admin_again", Method: "POST", Path: "/api/v1/setup/admin", Body: m{"full_name": "Admin User", "email": adminEmail, "password": password}},
		{Name: "setup_status_after", Method: "GET", Path: "/api/v1/setup/status"},
		{Name: "setup_user_without_token", Method: "POST", Path: "/api/v1/setup/user", Body: m{"full_name": "Demo User", "email": demoEmail, "password": password}},
		{Name: "setup_user", Method: "POST", Path: "/api/v1/setup/user", Auth: "admin_token",
			Body:    m{"full_name": "Demo User", "email": demoEmail, "password": password},
			Capture: m2("demo_id", "user_id")},
		{Name: "setup_user_conflict", Method: "POST", Path: "/api/v1/setup/user", Auth: "admin_token", Body: m{"full_name": "Demo User", "email": demoEmail, "password": password}},
		{Name: "setup_project", Method: "POST", Path: "/api/v1/setup/project", Auth: "admin_token",
			Body: m{"name": "Setup Project", "description": "from the wizard", "add_user_id": "{demo_id}"}},
		{Name: "setup_seed", Method: "POST", Path: "/api/v1/setup/seed", Auth: "admin_token"},

		// ---- authentication ----
		{Name: "login_wrong_password", Method: "POST", Path: "/api/v1/auth/login", Body: m{"email": demoEmail, "password": "Wrong1234"}},
		{Name: "login_unknown_user", Method: "POST", Path: "/api/v1/auth/login", Body: m{"email": "nobody@example.com", "password": password}},
		{Name: "login_validation", Method: "POST", Path: "/api/v1/auth/login", Body: m{"email": "not-an-email"}},
		{Name: "login_demo", Method: "POST", Path: "/api/v1/auth/login", Body: m{"email": demoEmail, "password": password},
			Capture: m2("demo_token", "access_token")},
		{Name: "refresh", Method: "POST", Path: "/api/v1/auth/refresh"},
		{Name: "logout", Method: "POST", Path: "/api/v1/auth/logout"},
		{Name: "refresh_after_logout", Method: "POST", Path: "/api/v1/auth/refresh"},
		{Name: "me", Method: "GET", Path: "/api/v1/auth/me", Auth: "admin_token"},
		{Name: "me_update", Method: "PUT", Path: "/api/v1/auth/me", Auth: "admin_token", Body: m{"full_name": "Admin Renamed", "timezone": "UTC"}},
		{Name: "me_update_bad_timezone", Method: "PUT", Path: "/api/v1/auth/me", Auth: "admin_token", Body: m{"timezone": "Not/AZone"}},
		{Name: "me_activity", Method: "GET", Path: "/api/v1/auth/me/activity", Auth: "admin_token"},
		{Name: "me_issues", Method: "GET", Path: "/api/v1/auth/me/issues", Auth: "admin_token"},
		{Name: "register_as_non_admin", Method: "POST", Path: "/api/v1/auth/register", Auth: "demo_token", Body: m{"full_name": "Third", "email": thirdEmail, "password": password}},
		{Name: "register", Method: "POST", Path: "/api/v1/auth/register", Auth: "admin_token",
			Body:    m{"full_name": "Third User", "email": thirdEmail, "password": password},
			Capture: m2("third_id", "user.id", "third_token", "access_token")},
		{Name: "register_conflict", Method: "POST", Path: "/api/v1/auth/register", Auth: "admin_token", Body: m{"full_name": "Third User", "email": thirdEmail, "password": password}},
		{Name: "register_invalid", Method: "POST", Path: "/api/v1/auth/register", Auth: "admin_token", Body: m{"full_name": "", "email": "x"}},
		{Name: "register_with_username", Method: "POST", Path: "/api/v1/auth/register", Auth: "admin_token", Body: m{"full_name": "Named User", "email": "named@example.com", "password": password, "username": "Named_User"}},
		{Name: "register_username_taken", Method: "POST", Path: "/api/v1/auth/register", Auth: "admin_token", Body: m{"full_name": "Other", "email": "other@example.com", "password": password, "username": "named_user"}},
		{Name: "register_username_reserved", Method: "POST", Path: "/api/v1/auth/register", Auth: "admin_token", Body: m{"full_name": "Other", "email": "other@example.com", "password": password, "username": "admin"}},
		{Name: "register_username_invalid", Method: "POST", Path: "/api/v1/auth/register", Auth: "admin_token", Body: m{"full_name": "Other", "email": "other@example.com", "password": password, "username": "no way"}},
		{Name: "username_check_available", Method: "GET", Path: "/api/v1/admin/usernames/check?username=fresh_name", Auth: "admin_token"},
		{Name: "username_check_taken", Method: "GET", Path: "/api/v1/admin/usernames/check?username=Named_User", Auth: "admin_token"},
		{Name: "username_check_non_admin", Method: "GET", Path: "/api/v1/admin/usernames/check?username=fresh_name", Auth: "demo_token"},
		{Name: "username_change_refused", Method: "PUT", Path: "/api/v1/auth/me", Auth: "admin_token", Body: m{"username": "someone_else"}},
		{Name: "username_change_refused_by_admin", Method: "PATCH", Path: "/api/v1/admin/users/{demo_id}", Auth: "admin_token", Body: m{"username": "someone_else"}},

		// ---- projects and members ----
		{Name: "projects_list", Method: "GET", Path: "/api/v1/projects", Auth: "admin_token"},
		{Name: "project_create_invalid", Method: "POST", Path: "/api/v1/projects", Auth: "admin_token", Body: m{}},
		{Name: "project_create", Method: "POST", Path: "/api/v1/projects", Auth: "admin_token", Body: m{"name": "Alpha", "key": "ALPHA", "description": "parity project"},
			Capture: m2("project_id", "id", "project_key", "key")},
		{Name: "project_create_duplicate_key", Method: "POST", Path: "/api/v1/projects", Auth: "admin_token", Body: m{"name": "Alpha 2", "key": "ALPHA"}},
		{Name: "project_get", Method: "GET", Path: "/api/v1/projects/{project_key}", Auth: "admin_token"},
		{Name: "project_get_not_found", Method: "GET", Path: "/api/v1/projects/nope", Auth: "admin_token"},
		{Name: "project_get_non_member", Method: "GET", Path: "/api/v1/projects/{project_key}", Auth: "third_token"},
		{Name: "project_update", Method: "PUT", Path: "/api/v1/projects/{project_key}", Auth: "admin_token", Body: m{"name": "Alpha Renamed", "description": "changed"}},
		{Name: "members_list", Method: "GET", Path: "/api/v1/projects/{project_key}/members", Auth: "admin_token"},
		{Name: "member_add_invalid", Method: "POST", Path: "/api/v1/projects/{project_key}/members", Auth: "admin_token", Body: m{"email": "x", "role": "boss"}},
		{Name: "member_add_demo", Method: "POST", Path: "/api/v1/projects/{project_key}/members", Auth: "admin_token", Body: m{"email": demoEmail, "role": "member"}},
		{Name: "member_add_unknown_email", Method: "POST", Path: "/api/v1/projects/{project_key}/members", Auth: "admin_token", Body: m{"email": "ghost@example.com", "role": "member"}},
		{Name: "member_add_by_non_admin", Method: "POST", Path: "/api/v1/projects/{project_key}/members", Auth: "demo_token", Body: m{"email": thirdEmail, "role": "member"}},
		{Name: "member_add_third", Method: "POST", Path: "/api/v1/projects/{project_key}/members", Auth: "admin_token", Body: m{"email": thirdEmail, "role": "viewer"}},
		{Name: "member_remove_third", Method: "DELETE", Path: "/api/v1/projects/{project_key}/members/{third_id}", Auth: "admin_token"},
		{Name: "members_list_after", Method: "GET", Path: "/api/v1/projects/{project_key}/members", Auth: "demo_token"},

		// ---- labels ----
		{Name: "label_create_invalid", Method: "POST", Path: "/api/v1/projects/{project_key}/labels", Auth: "admin_token", Body: m{}},
		{Name: "label_create_bad_color", Method: "POST", Path: "/api/v1/projects/{project_key}/labels", Auth: "admin_token", Body: m{"name": "x", "color": "#123456"}},
		{Name: "label_create", Method: "POST", Path: "/api/v1/projects/{project_key}/labels", Auth: "admin_token", Body: m{"name": "bug", "color": "#e11d48"},
			Capture: m2("label_id", "id")},
		{Name: "labels_list", Method: "GET", Path: "/api/v1/projects/{project_key}/labels", Auth: "demo_token"},

		// ---- issues ----
		{Name: "issue_create_invalid", Method: "POST", Path: "/api/v1/projects/{project_key}/issues", Auth: "admin_token", Body: m{"title": "x", "type": "nope", "priority": "low"}},
		{Name: "issue_create_one", Method: "POST", Path: "/api/v1/projects/{project_key}/issues", Auth: "admin_token",
			Body:    m{"title": "First issue", "description": "hello", "type": "task", "priority": "high", "story_points": 3},
			Capture: m2("issue1_id", "id", "issue1_key", "key")},
		{Name: "issue_create_two", Method: "POST", Path: "/api/v1/projects/{project_key}/issues", Auth: "demo_token",
			Body:    m{"title": "Second issue", "type": "bug", "priority": "medium", "assignee_id": "{admin_id}"},
			Capture: m2("issue2_id", "id")},
		{Name: "issues_list", Method: "GET", Path: "/api/v1/projects/{project_key}/issues", Auth: "admin_token"},
		{Name: "issues_list_filtered", Method: "GET", Path: "/api/v1/projects/{project_key}/issues?q=First&type=task&page=1&limit=5", Auth: "admin_token"},
		{Name: "issue_get", Method: "GET", Path: "/api/v1/projects/{project_key}/issues/{issue1_id}", Auth: "admin_token"},
		{Name: "issue_get_not_found", Method: "GET", Path: "/api/v1/projects/{project_key}/issues/99999", Auth: "admin_token"},
		{Name: "issue_get_bad_id", Method: "GET", Path: "/api/v1/projects/{project_key}/issues/abc", Auth: "admin_token"},
		{Name: "issue_update", Method: "PUT", Path: "/api/v1/projects/{project_key}/issues/{issue1_id}", Auth: "admin_token", Body: m{"status": "in_progress", "priority": "critical"}},
		{Name: "issue_update_invalid", Method: "PUT", Path: "/api/v1/projects/{project_key}/issues/{issue1_id}", Auth: "admin_token", Body: m{"status": "nope"}},
		{Name: "issue_update_blank_title", Method: "PUT", Path: "/api/v1/projects/{project_key}/issues/{issue1_id}", Auth: "admin_token", Body: m{"title": "  "}},
		{Name: "issue_assign_clear", Method: "PUT", Path: "/api/v1/projects/{project_key}/issues/{issue2_id}", Auth: "admin_token", Body: m{"assignee_id": 0}},
		{Name: "board", Method: "GET", Path: "/api/v1/projects/{project_key}/board", Auth: "admin_token"},
		{Name: "backlog", Method: "GET", Path: "/api/v1/projects/{project_key}/backlog", Auth: "admin_token"},
		{Name: "velocity", Method: "GET", Path: "/api/v1/projects/{project_key}/velocity", Auth: "admin_token"},

		// ---- sprints ----
		{Name: "sprint_create_invalid", Method: "POST", Path: "/api/v1/projects/{project_key}/sprints", Auth: "admin_token", Body: m{}},
		{Name: "sprint_create", Method: "POST", Path: "/api/v1/projects/{project_key}/sprints", Auth: "admin_token",
			Body:    m{"name": "Sprint One", "goal": "ship", "start_date": "2030-01-01T00:00:00Z", "end_date": "2030-01-14T00:00:00Z"},
			Capture: m2("sprint_id", "id")},
		{Name: "sprint_create_too_long", Method: "POST", Path: "/api/v1/projects/{project_key}/sprints", Auth: "admin_token",
			Body: m{"name": "Too long", "start_date": "2030-01-01T00:00:00Z", "end_date": "2030-03-01T00:00:00Z"}},
		{Name: "sprint_create_by_member", Method: "POST", Path: "/api/v1/projects/{project_key}/sprints", Auth: "demo_token", Body: m{"name": "Member sprint"}},
		{Name: "sprints_list", Method: "GET", Path: "/api/v1/projects/{project_key}/sprints", Auth: "admin_token"},
		{Name: "sprint_get", Method: "GET", Path: "/api/v1/projects/{project_key}/sprints/{sprint_id}", Auth: "admin_token"},
		{Name: "sprint_update", Method: "PUT", Path: "/api/v1/projects/{project_key}/sprints/{sprint_id}", Auth: "admin_token", Body: m{"goal": "ship it"}},
		{Name: "sprint_add_issue", Method: "PUT", Path: "/api/v1/projects/{project_key}/issues/{issue1_id}", Auth: "demo_token", Body: m{"sprint_id": "{sprint_id}"}},
		{Name: "sprint_issues", Method: "GET", Path: "/api/v1/projects/{project_key}/sprints/{sprint_id}/issues", Auth: "admin_token"},
		{Name: "sprint_start", Method: "POST", Path: "/api/v1/projects/{project_key}/sprints/{sprint_id}/start", Auth: "admin_token"},
		{Name: "sprint_start_again", Method: "POST", Path: "/api/v1/projects/{project_key}/sprints/{sprint_id}/start", Auth: "admin_token"},
		{Name: "sprint_delete_active", Method: "DELETE", Path: "/api/v1/projects/{project_key}/sprints/{sprint_id}", Auth: "admin_token"},
		{Name: "sprint_burndown", Method: "GET", Path: "/api/v1/projects/{project_key}/sprints/{sprint_id}/burndown", Auth: "admin_token"},
		{Name: "sprint_complete", Method: "POST", Path: "/api/v1/projects/{project_key}/sprints/{sprint_id}/complete", Auth: "admin_token", Body: m{}},
		{Name: "sprint_update_completed", Method: "PUT", Path: "/api/v1/projects/{project_key}/sprints/{sprint_id}", Auth: "admin_token", Body: m{"goal": "again"}},
		{Name: "sprint_create_second", Method: "POST", Path: "/api/v1/projects/{project_key}/sprints", Auth: "admin_token", Body: m{"name": "Sprint Two"},
			Capture: m2("sprint2_id", "id")},
		{Name: "sprint_delete", Method: "DELETE", Path: "/api/v1/projects/{project_key}/sprints/{sprint2_id}", Auth: "admin_token"},
		{Name: "sprint_get_not_found", Method: "GET", Path: "/api/v1/projects/{project_key}/sprints/99999", Auth: "admin_token"},
		{Name: "velocity_after", Method: "GET", Path: "/api/v1/projects/{project_key}/velocity", Auth: "admin_token"},

		// ---- comments, activity, notifications ----
		{Name: "comment_create_invalid", Method: "POST", Path: "/api/v1/issues/{issue1_id}/comments", Auth: "admin_token", Body: m{}},
		{Name: "comment_create", Method: "POST", Path: "/api/v1/issues/{issue1_id}/comments", Auth: "admin_token", Body: m{"body": "hello @demo please look"},
			Capture: m2("comment_id", "id")},
		{Name: "comment_create_by_demo", Method: "POST", Path: "/api/v1/issues/{issue1_id}/comments", Auth: "demo_token", Body: m{"body": "thanks"}, Capture: m2("comment2_id", "id")},
		{Name: "comments_list", Method: "GET", Path: "/api/v1/issues/{issue1_id}/comments", Auth: "admin_token"},
		{Name: "comment_update", Method: "PUT", Path: "/api/v1/issues/{issue1_id}/comments/{comment_id}", Auth: "admin_token", Body: m{"body": "edited"}},
		{Name: "comment_update_other_users", Method: "PUT", Path: "/api/v1/issues/{issue1_id}/comments/{comment_id}", Auth: "demo_token", Body: m{"body": "hijack"}},
		{Name: "comment_delete", Method: "DELETE", Path: "/api/v1/issues/{issue1_id}/comments/{comment2_id}", Auth: "demo_token"},
		{Name: "activity", Method: "GET", Path: "/api/v1/issues/{issue1_id}/activity", Auth: "admin_token"},
		{Name: "activity_non_member", Method: "GET", Path: "/api/v1/issues/{issue1_id}/activity", Auth: "third_token"},
		{Name: "notifications_demo", Method: "GET", Path: "/api/v1/notifications", Auth: "demo_token"},
		{Name: "notification_mark_read", Method: "PUT", Path: "/api/v1/notifications/1/read", Auth: "demo_token"},
		{Name: "notification_mark_read_missing", Method: "PUT", Path: "/api/v1/notifications/99999/read", Auth: "demo_token"},

		// ---- labels on issues, relations ----
		{Name: "issue_label_attach", Method: "POST", Path: "/api/v1/issues/{issue1_id}/labels/{label_id}", Auth: "admin_token"},
		{Name: "issue_labels_list", Method: "GET", Path: "/api/v1/issues/{issue1_id}/labels", Auth: "admin_token"},
		{Name: "issue_label_detach", Method: "DELETE", Path: "/api/v1/issues/{issue1_id}/labels/{label_id}", Auth: "admin_token"},
		{Name: "relation_create_self", Method: "POST", Path: "/api/v1/issues/{issue1_id}/relations", Auth: "admin_token", Body: m{"target_issue_id": "{issue1_id}", "relation_type": "blocks"}},
		{Name: "relation_create", Method: "POST", Path: "/api/v1/issues/{issue1_id}/relations", Auth: "admin_token", Body: m{"target_issue_id": "{issue2_id}", "relation_type": "blocks"},
			Capture: m2("relation_id", "id")},
		{Name: "relation_create_duplicate", Method: "POST", Path: "/api/v1/issues/{issue1_id}/relations", Auth: "admin_token", Body: m{"target_issue_id": "{issue2_id}", "relation_type": "blocks"}},
		{Name: "relations_list", Method: "GET", Path: "/api/v1/issues/{issue1_id}/relations", Auth: "admin_token"},
		{Name: "relations_list_non_member", Method: "GET", Path: "/api/v1/issues/{issue1_id}/relations", Auth: "third_token"},
		{Name: "relation_create_non_member", Method: "POST", Path: "/api/v1/issues/{issue1_id}/relations", Auth: "third_token", Body: m{"target_issue_id": "{issue2_id}", "relation_type": "relates_to"}},
		{Name: "relation_delete_non_member", Method: "DELETE", Path: "/api/v1/issues/{issue1_id}/relations/{relation_id}", Auth: "third_token"},
		{Name: "relation_delete_wrong_issue", Method: "DELETE", Path: "/api/v1/issues/{issue2_id}/relations/99999", Auth: "admin_token"},
		{Name: "relation_delete", Method: "DELETE", Path: "/api/v1/issues/{issue1_id}/relations/{relation_id}", Auth: "admin_token"},
		{Name: "label_delete", Method: "DELETE", Path: "/api/v1/projects/{project_key}/labels/{label_id}", Auth: "admin_token"},

		// ---- attachments (listing needs no file) ----
		{Name: "attachments_list", Method: "GET", Path: "/api/v1/issues/{issue1_id}/attachments", Auth: "admin_token"},
		{Name: "attachment_upload_no_file", Method: "POST", Path: "/api/v1/issues/{issue1_id}/attachments", Auth: "admin_token", Raw: []byte(`{}`)},
		{Name: "attachment_download_missing", Method: "GET", Path: "/api/v1/issues/{issue1_id}/attachments/99999/download", Auth: "admin_token"},

		// ---- settings and admin ----
		{Name: "settings_get", Method: "GET", Path: "/api/v1/settings", Auth: "admin_token"},
		{Name: "settings_patch_by_non_admin", Method: "PATCH", Path: "/api/v1/settings", Auth: "demo_token", Body: m{"organization_name": "Nope"}},
		{Name: "settings_patch", Method: "PATCH", Path: "/api/v1/settings", Auth: "admin_token", Body: m{"organization_name": "Parity Org", "date_format": "DD/MM/YYYY"}},
		{Name: "settings_patch_invalid", Method: "PATCH", Path: "/api/v1/settings", Auth: "admin_token", Body: m{"date_format": "nonsense"}},
		{Name: "admin_users_list", Method: "GET", Path: "/api/v1/admin/users", Auth: "admin_token"},
		{Name: "admin_users_list_non_admin", Method: "GET", Path: "/api/v1/admin/users", Auth: "demo_token"},
		{Name: "admin_user_patch_empty", Method: "PATCH", Path: "/api/v1/admin/users/{demo_id}", Auth: "admin_token", Body: m{}},
		{Name: "admin_user_deactivate", Method: "PATCH", Path: "/api/v1/admin/users/{third_id}", Auth: "admin_token", Body: m{"is_active": false}},
		{Name: "admin_user_demote_last_admin", Method: "PATCH", Path: "/api/v1/admin/users/{admin_id}", Auth: "admin_token", Body: m{"is_admin": false}},
		{Name: "admin_user_not_found", Method: "PATCH", Path: "/api/v1/admin/users/99999", Auth: "admin_token", Body: m{"is_active": false}},
		{Name: "system_logs", Method: "GET", Path: "/api/v1/admin/system-logs?size=5", Auth: "admin_token"},
		{Name: "system_logs_invalid_category", Method: "GET", Path: "/api/v1/admin/system-logs?category=nope", Auth: "admin_token"},
		{Name: "system_logs_non_admin", Method: "GET", Path: "/api/v1/admin/system-logs", Auth: "demo_token"},

		// ---- password flows ----
		{Name: "password_change_wrong_current", Method: "PUT", Path: "/api/v1/auth/me/password", Auth: "demo_token", Body: m{"current_password": "Wrong1234", "new_password": "Newpassw0rd!"}},
		{Name: "password_change_same", Method: "PUT", Path: "/api/v1/auth/me/password", Auth: "demo_token", Body: m{"current_password": password, "new_password": password}},
		{Name: "password_change", Method: "PUT", Path: "/api/v1/auth/me/password", Auth: "demo_token", Body: m{"current_password": password, "new_password": "Newpassw0rd!"}},
		{Name: "forgot_password_unknown", Method: "POST", Path: "/api/v1/auth/forgot-password", Body: m{"email": "ghost@example.com"}},
		{Name: "forgot_password", Method: "POST", Path: "/api/v1/auth/forgot-password", Body: m{"email": demoEmail}, Capture: m2("reset_token", "token")},
		{Name: "reset_password_bad_token", Method: "POST", Path: "/api/v1/auth/reset-password", Body: m{"token": strings.Repeat("a", 64), "password": "Another1234"}},
		{Name: "reset_password", Method: "POST", Path: "/api/v1/auth/reset-password", Body: m{"token": "{reset_token}", "password": "Another1234"}},
		{Name: "login_after_reset", Method: "POST", Path: "/api/v1/auth/login", Body: m{"email": demoEmail, "password": "Another1234"}},
		{Name: "login_deactivated", Method: "POST", Path: "/api/v1/auth/login", Body: m{"email": thirdEmail, "password": password}},

		// ---- deletes ----
		{Name: "issue_delete", Method: "DELETE", Path: "/api/v1/projects/{project_key}/issues/{issue2_id}", Auth: "admin_token"},
		{Name: "issue_delete_missing", Method: "DELETE", Path: "/api/v1/projects/{project_key}/issues/{issue2_id}", Auth: "admin_token"},
		{Name: "project_delete_by_non_owner", Method: "DELETE", Path: "/api/v1/projects/{project_key}", Auth: "demo_token"},
		{Name: "project_delete", Method: "DELETE", Path: "/api/v1/projects/{project_key}", Auth: "admin_token"},
		{Name: "project_get_deleted", Method: "GET", Path: "/api/v1/projects/{project_key}", Auth: "admin_token"},
	}
	return s
}

func m2(kv ...string) map[string]string {
	out := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		out[kv[i]] = kv[i+1]
	}
	return out
}
