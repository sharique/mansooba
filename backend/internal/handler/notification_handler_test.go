package handler_test

// Real SQLite and the real repository, no mocks: the list is scoped to the caller
// and mark-read must refuse another user's row (IDOR guard).

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/sharique/mansooba/internal/domain"
	"github.com/sharique/mansooba/internal/dto"
	"github.com/sharique/mansooba/internal/handler"
	"github.com/sharique/mansooba/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type notifHandlerEnv struct {
	db     *gorm.DB
	h      *handler.NotificationHandler
	alice  uint
	bob    uint
	projID uint
	issue  uint
}

func newNotifHandlerEnv(t *testing.T) *notifHandlerEnv {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.User{}, &domain.Project{}, &domain.Issue{}, &domain.Notification{}))
	alice := &domain.User{Name: "Alice", Email: "alice@x.com", Username: "alice", Password: "h", IsActive: true}
	bob := &domain.User{Name: "Bob", Email: "bob@x.com", Username: "bob", Password: "h", IsActive: true}
	require.NoError(t, db.Create(alice).Error)
	require.NoError(t, db.Create(bob).Error)
	p := &domain.Project{Key: "apol", Name: "Apollo", OwnerID: alice.ID}
	require.NoError(t, db.Create(p).Error)
	issue := &domain.Issue{Key: "APOL-1", ProjectID: p.ID, Title: "T", Type: domain.IssueTypeTask, Status: domain.IssueStatusTodo, Priority: domain.IssuePriorityLow, ReporterID: alice.ID}
	require.NoError(t, db.Create(issue).Error)
	return &notifHandlerEnv{db: db, h: handler.NewNotificationHandler(repository.NewNotificationRepository(db)), alice: alice.ID, bob: bob.ID, projID: p.ID, issue: issue.ID}
}

func (e *notifHandlerEnv) call(t *testing.T, method, path string, userID uint, id string, fn func(*echo.Context) error) *httptest.ResponseRecorder {
	t.Helper()
	ec := echo.New()
	req := httptest.NewRequest(method, path, nil)
	rec := httptest.NewRecorder()
	c := ec.NewContext(req, rec)
	c.Set("userID", userID)
	if id != "" {
		setParams(c, []string{"id"}, id)
	}
	if err := fn(c); err != nil {
		ec.HTTPErrorHandler(c, err)
	}
	return rec
}

func TestNotificationHandler_List_ReturnsNewFieldsForCallerOnly(t *testing.T) {
	e := newNotifHandlerEnv(t)
	require.NoError(t, e.db.Create(&domain.Notification{RecipientID: e.bob, ActorID: e.alice, Kind: domain.NotificationKindTaskAssigned, IssueID: e.issue}).Error)
	require.NoError(t, e.db.Create(&domain.Notification{RecipientID: e.bob, ActorID: e.alice, Kind: domain.NotificationKindProjectMemberRemoved, ProjectID: e.projID}).Error)
	require.NoError(t, e.db.Create(&domain.Notification{RecipientID: e.alice, ActorID: e.bob, Kind: domain.NotificationKindTaskAssigned, IssueID: e.issue}).Error)

	rec := e.call(t, http.MethodGet, "/notifications", e.bob, "", e.h.List)
	require.Equal(t, http.StatusOK, rec.Code)
	var got []dto.NotificationResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got, 2)
	for _, n := range got {
		assert.Equal(t, e.bob, n.RecipientID, "must never include another user's row")
		assert.Equal(t, "Alice", n.ActorName)
		assert.Equal(t, "Apollo", n.ProjectName)
		assert.Equal(t, "apol", n.ProjectKey)
	}
	kinds := []string{got[0].Kind, got[1].Kind}
	assert.ElementsMatch(t, []string{domain.NotificationKindTaskAssigned, domain.NotificationKindProjectMemberRemoved}, kinds)
}

func TestNotificationHandler_MarkRead_OtherUsersNotificationIs404AndStaysUnread(t *testing.T) {
	e := newNotifHandlerEnv(t)
	n := &domain.Notification{RecipientID: e.alice, ActorID: e.bob, Kind: domain.NotificationKindTaskAssigned, IssueID: e.issue}
	require.NoError(t, e.db.Create(n).Error)

	rec := e.call(t, http.MethodPut, "/notifications/x/read", e.bob, strconv.Itoa(int(n.ID)), e.h.MarkRead)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	var after domain.Notification
	require.NoError(t, e.db.First(&after, n.ID).Error)
	assert.False(t, after.Read)

	rec = e.call(t, http.MethodPut, "/notifications/x/read", e.alice, strconv.Itoa(int(n.ID)), e.h.MarkRead)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

// Seeded projects are stored with an upper-case key (DEMO) while every project
// API returns it lower-cased; the notification must agree with the project list
// or the client cannot tell the project is the user's own.
func TestNotificationHandler_List_ProjectKeyIsLowerCasedLikeTheProjectAPI(t *testing.T) {
	e := newNotifHandlerEnv(t)
	require.NoError(t, e.db.Model(&domain.Project{}).Where("id = ?", e.projID).Update("key", "DEMO").Error)
	require.NoError(t, e.db.Create(&domain.Notification{RecipientID: e.bob, ActorID: e.alice, Kind: domain.NotificationKindTaskAssigned, IssueID: e.issue}).Error)

	rec := e.call(t, http.MethodGet, "/notifications", e.bob, "", e.h.List)
	var got []dto.NotificationResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Len(t, got, 1)
	assert.Equal(t, "demo", got[0].ProjectKey)
}

