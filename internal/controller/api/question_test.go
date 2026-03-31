package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/syt3s/TreeBox/internal/config"
	"github.com/syt3s/TreeBox/internal/dbutil"
	appcontext "github.com/syt3s/TreeBox/internal/http/appctx"
	"github.com/syt3s/TreeBox/internal/http/middleware"
	"github.com/syt3s/TreeBox/internal/model"
	"github.com/syt3s/TreeBox/internal/notify"
	"github.com/syt3s/TreeBox/internal/repository"
)

func TestCreateQuestionSkipsRecaptchaOutsideProduction(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldApp := config.App
	oldUsers := repository.Users
	oldQuestions := repository.Questions
	oldHTTPClient := http.DefaultClient
	t.Cleanup(func() {
		config.App = oldApp
		repository.Users = oldUsers
		repository.Questions = oldQuestions
		http.DefaultClient = oldHTTPClient
	})

	config.App.Production = false

	repository.Users = &stubUserRepository{
		userByDomain: &model.User{
			Model:  gorm.Model{ID: 42},
			Domain: "nailong",
		},
	}

	questionsRepo := &stubQuestionRepository{}
	repository.Questions = questionsRepo

	http.DefaultClient = &http.Client{
		Transport: roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("recaptcha verification should be skipped outside production")
		}),
	}

	engine := gin.New()
	engine.Use(appcontext.Contexter(), middleware.ErrorHandler())

	apiRoutes := engine.Group("/api/v2")
	apiRoutes.Use(appcontext.APIEndpoint())
	apiRoutes.POST("/questions/:domain", appcontext.Wrap(CreateQuestion))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v2/questions/nailong",
		strings.NewReader(`{"content":"hello","receive_reply_email":"asker@example.com","recaptcha":"test"}`),
	)
	request.Header.Set("Content-Type", "application/json")

	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Len(t, questionsRepo.createCalls, 1)
	require.Equal(t, "hello", questionsRepo.createCalls[0].Content)
	require.Equal(t, "asker@example.com", questionsRepo.createCalls[0].ReceiveReplyEmail)

	var response struct {
		Code int `json:"code"`
		Data struct {
			Success  bool            `json:"success"`
			Message  string          `json:"message"`
			Question *model.Question `json:"question"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	require.True(t, response.Data.Success)
	require.NotNil(t, response.Data.Question)
	require.Equal(t, "hello", response.Data.Question.Content)
}

func TestAnswerQuestionSendsReplyEmailOnFirstAnswer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldApp := config.App
	oldUsers := repository.Users
	oldQuestions := repository.Questions
	t.Cleanup(func() {
		config.App = oldApp
		repository.Users = oldUsers
		repository.Questions = oldQuestions
	})

	config.App.ExternalURL = "http://frontend.local"

	pageUser := &model.User{
		Model:  gorm.Model{ID: 42},
		Name:   "Tree Owner",
		Domain: "nailong",
	}

	repository.Users = &stubUserRepository{userByDomain: pageUser}
	questionsRepo := &stubQuestionRepository{
		questionsByID: map[uint]*model.Question{
			1: {
				Model:             dbutil.Model{ID: 1},
				UserID:            pageUser.ID,
				Content:           "hello",
				ReceiveReplyEmail: "asker@example.com",
			},
		},
	}
	repository.Questions = questionsRepo

	replySender := &stubReplyEmailSender{}
	restoreSender := notify.SetReplyEmailSender(replySender)
	t.Cleanup(restoreSender)

	engine := gin.New()
	engine.Use(appcontext.Contexter(), middleware.ErrorHandler(), testAuthMiddleware(pageUser))

	apiRoutes := engine.Group("/api/v2")
	apiRoutes.Use(appcontext.APIEndpoint())
	apiRoutes.POST("/questions/:domain/:questionID/answer", appcontext.Wrap(AnswerQuestion))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v2/questions/nailong/1/answer",
		strings.NewReader(`{"answer":"world"}`),
	)
	request.Header.Set("Content-Type", "application/json")

	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "world", questionsRepo.questionsByID[1].Answer)
	require.Len(t, replySender.calls, 1)
	require.Equal(t, "asker@example.com", replySender.calls[0].To)
	require.Equal(t, "hello", replySender.calls[0].QuestionContent)
	require.Equal(t, "world", replySender.calls[0].AnswerContent)
	require.Equal(t, "http://frontend.local/box/nailong/1", replySender.calls[0].QuestionURL)
}

func TestAnswerQuestionDoesNotResendReplyEmailForAlreadyAnsweredQuestion(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldApp := config.App
	oldUsers := repository.Users
	oldQuestions := repository.Questions
	t.Cleanup(func() {
		config.App = oldApp
		repository.Users = oldUsers
		repository.Questions = oldQuestions
	})

	pageUser := &model.User{
		Model:  gorm.Model{ID: 42},
		Name:   "Tree Owner",
		Domain: "nailong",
	}

	repository.Users = &stubUserRepository{userByDomain: pageUser}
	questionsRepo := &stubQuestionRepository{
		questionsByID: map[uint]*model.Question{
			1: {
				Model:             dbutil.Model{ID: 1},
				UserID:            pageUser.ID,
				Content:           "hello",
				Answer:            "existing answer",
				ReceiveReplyEmail: "asker@example.com",
			},
		},
	}
	repository.Questions = questionsRepo

	replySender := &stubReplyEmailSender{}
	restoreSender := notify.SetReplyEmailSender(replySender)
	t.Cleanup(restoreSender)

	engine := gin.New()
	engine.Use(appcontext.Contexter(), middleware.ErrorHandler(), testAuthMiddleware(pageUser))

	apiRoutes := engine.Group("/api/v2")
	apiRoutes.Use(appcontext.APIEndpoint())
	apiRoutes.POST("/questions/:domain/:questionID/answer", appcontext.Wrap(AnswerQuestion))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v2/questions/nailong/1/answer",
		strings.NewReader(`{"answer":"updated answer"}`),
	)
	request.Header.Set("Content-Type", "application/json")

	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "updated answer", questionsRepo.questionsByID[1].Answer)
	require.Empty(t, replySender.calls)
}

func TestAnswerQuestionStillSucceedsWhenReplyEmailFails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldApp := config.App
	oldUsers := repository.Users
	oldQuestions := repository.Questions
	t.Cleanup(func() {
		config.App = oldApp
		repository.Users = oldUsers
		repository.Questions = oldQuestions
	})

	pageUser := &model.User{
		Model:  gorm.Model{ID: 42},
		Name:   "Tree Owner",
		Domain: "nailong",
	}

	repository.Users = &stubUserRepository{userByDomain: pageUser}
	questionsRepo := &stubQuestionRepository{
		questionsByID: map[uint]*model.Question{
			1: {
				Model:             dbutil.Model{ID: 1},
				UserID:            pageUser.ID,
				Content:           "hello",
				ReceiveReplyEmail: "asker@example.com",
			},
		},
	}
	repository.Questions = questionsRepo

	replySender := &stubReplyEmailSender{err: errors.New("smtp failed")}
	restoreSender := notify.SetReplyEmailSender(replySender)
	t.Cleanup(restoreSender)

	engine := gin.New()
	engine.Use(appcontext.Contexter(), middleware.ErrorHandler(), testAuthMiddleware(pageUser))

	apiRoutes := engine.Group("/api/v2")
	apiRoutes.Use(appcontext.APIEndpoint())
	apiRoutes.POST("/questions/:domain/:questionID/answer", appcontext.Wrap(AnswerQuestion))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v2/questions/nailong/1/answer",
		strings.NewReader(`{"answer":"world"}`),
	)
	request.Header.Set("Content-Type", "application/json")

	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "world", questionsRepo.questionsByID[1].Answer)
	require.Len(t, replySender.calls, 1)
}

func TestGetUserQuestionStats(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldQuestions := repository.Questions
	t.Cleanup(func() {
		repository.Questions = oldQuestions
	})

	repository.Questions = &stubQuestionRepository{
		totalCount:    7,
		answeredCount: 3,
		unreadCount:   2,
	}

	engine := gin.New()
	engine.Use(appcontext.Contexter(), middleware.ErrorHandler(), testAuthMiddleware(&model.User{
		Model: gorm.Model{ID: 42},
	}))

	apiRoutes := engine.Group("/api/v2")
	apiRoutes.Use(appcontext.APIEndpoint())
	apiRoutes.GET("/user/questions/stats", appcontext.Wrap(GetUserQuestionStats))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v2/user/questions/stats", nil)

	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Code int `json:"code"`
		Data struct {
			Success       bool  `json:"success"`
			TotalCount    int64 `json:"total_count"`
			AnsweredCount int64 `json:"answered_count"`
			UnreadCount   int64 `json:"unread_count"`
			PendingCount  int64 `json:"pending_count"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	require.True(t, response.Data.Success)
	require.Equal(t, int64(7), response.Data.TotalCount)
	require.Equal(t, int64(3), response.Data.AnsweredCount)
	require.Equal(t, int64(2), response.Data.UnreadCount)
	require.Equal(t, int64(2), response.Data.PendingCount)
}

func TestGetUserQuestionStatsErrorMessageIsReadable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldQuestions := repository.Questions
	t.Cleanup(func() {
		repository.Questions = oldQuestions
	})

	repository.Questions = &stubQuestionRepository{
		totalCount:     7,
		answeredCount:  3,
		unreadCountErr: errors.New("count unread failed"),
	}

	engine := gin.New()
	engine.Use(appcontext.Contexter(), middleware.ErrorHandler(), testAuthMiddleware(&model.User{
		Model: gorm.Model{ID: 42},
	}))

	apiRoutes := engine.Group("/api/v2")
	apiRoutes.Use(appcontext.APIEndpoint())
	apiRoutes.GET("/user/questions/stats", appcontext.Wrap(GetUserQuestionStats))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v2/user/questions/stats", nil)

	engine.ServeHTTP(recorder, request)

	requireAPIError(t, recorder, http.StatusInternalServerError, 50000, "获取问题统计失败")
}

func TestMarkUserQuestionViewed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldQuestions := repository.Questions
	t.Cleanup(func() {
		repository.Questions = oldQuestions
	})

	repository.Questions = &stubQuestionRepository{
		questionsByID: map[uint]*model.Question{
			1: {
				Model:  dbutil.Model{ID: 1},
				UserID: 42,
			},
		},
	}

	engine := gin.New()
	engine.Use(appcontext.Contexter(), middleware.ErrorHandler(), testAuthMiddleware(&model.User{
		Model: gorm.Model{ID: 42},
	}))

	apiRoutes := engine.Group("/api/v2")
	apiRoutes.Use(appcontext.APIEndpoint())
	apiRoutes.POST("/user/questions/:questionID/viewed", appcontext.Wrap(MarkUserQuestionViewed))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v2/user/questions/1/viewed", nil)

	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Code int `json:"code"`
		Data struct {
			Success  bool   `json:"success"`
			ViewedAt string `json:"viewed_at"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	require.True(t, response.Data.Success)
	require.NotEmpty(t, response.Data.ViewedAt)
}

func TestMarkUserQuestionViewedErrorMessagesAreReadable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name       string
		path       string
		repository *stubQuestionRepository
		statusCode int
		errorCode  int
		message    string
	}{
		{
			name:       "invalid question id",
			path:       "/api/v2/user/questions/not-a-number/viewed",
			repository: &stubQuestionRepository{},
			statusCode: http.StatusBadRequest,
			errorCode:  40000,
			message:    "问题编号无效",
		},
		{
			name:       "question not found",
			path:       "/api/v2/user/questions/1/viewed",
			repository: &stubQuestionRepository{},
			statusCode: http.StatusNotFound,
			errorCode:  40400,
			message:    "问题不存在",
		},
		{
			name:       "get question failed",
			path:       "/api/v2/user/questions/1/viewed",
			repository: &stubQuestionRepository{getByIDErr: errors.New("get question failed")},
			statusCode: http.StatusInternalServerError,
			errorCode:  50000,
			message:    "获取问题失败",
		},
		{
			name: "forbidden",
			path: "/api/v2/user/questions/1/viewed",
			repository: &stubQuestionRepository{
				questionsByID: map[uint]*model.Question{
					1: {
						Model:  dbutil.Model{ID: 1},
						UserID: 99,
					},
				},
			},
			statusCode: http.StatusForbidden,
			errorCode:  40300,
			message:    "无权操作该问题",
		},
		{
			name: "mark viewed failed",
			path: "/api/v2/user/questions/1/viewed",
			repository: &stubQuestionRepository{
				questionsByID: map[uint]*model.Question{
					1: {
						Model:  dbutil.Model{ID: 1},
						UserID: 42,
					},
				},
				markViewedErr: errors.New("mark viewed failed"),
			},
			statusCode: http.StatusInternalServerError,
			errorCode:  50000,
			message:    "标记已查看失败",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			oldQuestions := repository.Questions
			t.Cleanup(func() {
				repository.Questions = oldQuestions
			})

			repository.Questions = testCase.repository

			engine := gin.New()
			engine.Use(appcontext.Contexter(), middleware.ErrorHandler(), testAuthMiddleware(&model.User{
				Model: gorm.Model{ID: 42},
			}))

			apiRoutes := engine.Group("/api/v2")
			apiRoutes.Use(appcontext.APIEndpoint())
			apiRoutes.POST("/user/questions/:questionID/viewed", appcontext.Wrap(MarkUserQuestionViewed))

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, testCase.path, nil)

			engine.ServeHTTP(recorder, request)

			requireAPIError(t, recorder, testCase.statusCode, testCase.errorCode, testCase.message)
		})
	}
}

func TestMarkAllUserQuestionsViewed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldQuestions := repository.Questions
	t.Cleanup(func() {
		repository.Questions = oldQuestions
	})

	repository.Questions = &stubQuestionRepository{
		questionsByID: map[uint]*model.Question{
			1: {
				Model:  dbutil.Model{ID: 1},
				UserID: 42,
			},
			2: {
				Model:  dbutil.Model{ID: 2},
				UserID: 42,
			},
			3: {
				Model:    dbutil.Model{ID: 3},
				UserID:   42,
				ViewedAt: ptrTime(time.Now().Add(-time.Hour)),
			},
			4: {
				Model:  dbutil.Model{ID: 4},
				UserID: 99,
			},
		},
	}

	engine := gin.New()
	engine.Use(appcontext.Contexter(), middleware.ErrorHandler(), testAuthMiddleware(&model.User{
		Model: gorm.Model{ID: 42},
	}))

	apiRoutes := engine.Group("/api/v2")
	apiRoutes.Use(appcontext.APIEndpoint())
	apiRoutes.POST("/user/questions/viewed", appcontext.Wrap(MarkAllUserQuestionsViewed))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v2/user/questions/viewed", nil)

	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Code int `json:"code"`
		Data struct {
			Success     bool   `json:"success"`
			ViewedAt    string `json:"viewed_at"`
			ViewedCount int64  `json:"viewed_count"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, 0, response.Code)
	require.True(t, response.Data.Success)
	require.Equal(t, int64(2), response.Data.ViewedCount)
	require.NotEmpty(t, response.Data.ViewedAt)
}

func TestMarkAllUserQuestionsViewedErrorMessageIsReadable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	oldQuestions := repository.Questions
	t.Cleanup(func() {
		repository.Questions = oldQuestions
	})

	repository.Questions = &stubQuestionRepository{
		markAllViewedErr: errors.New("mark all viewed failed"),
	}

	engine := gin.New()
	engine.Use(appcontext.Contexter(), middleware.ErrorHandler(), testAuthMiddleware(&model.User{
		Model: gorm.Model{ID: 42},
	}))

	apiRoutes := engine.Group("/api/v2")
	apiRoutes.Use(appcontext.APIEndpoint())
	apiRoutes.POST("/user/questions/viewed", appcontext.Wrap(MarkAllUserQuestionsViewed))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v2/user/questions/viewed", nil)

	engine.ServeHTTP(recorder, request)

	requireAPIError(t, recorder, http.StatusInternalServerError, 50000, "标记全部问题已查看失败")
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type stubUserRepository struct {
	userByDomain *model.User
	userByEmail  map[string]*model.User
	userByID     map[uint]*model.User
}

func (s *stubUserRepository) Create(context.Context, repository.CreateUserOptions) error {
	return errors.New("not implemented")
}

func (s *stubUserRepository) Register(context.Context, repository.RegisterUserOptions) (*repository.RegisterUserResult, error) {
	return nil, errors.New("not implemented")
}

func (s *stubUserRepository) EnsureTenantBootstrap(_ context.Context, userID uint) (*repository.TenantBootstrapResult, error) {
	if s.userByDomain != nil && s.userByDomain.ID == userID {
		return &repository.TenantBootstrapResult{
			User: s.userByDomain,
			Tenant: &model.Tenant{
				Model:       dbutil.Model{ID: 1},
				UID:         "tenant_1",
				Name:        "Test tenant",
				OwnerUserID: userID,
				IsPersonal:  true,
			},
			Membership: &model.TenantMember{
				TenantID: 1,
				UserID:   userID,
				Role:     model.TenantRoleOwner,
			},
			Workspace: &model.Workspace{
				Model:           dbutil.Model{ID: 1},
				UID:             "workspace_1",
				TenantID:        1,
				Name:            "Default workspace",
				CreatedByUserID: userID,
				IsDefault:       true,
			},
		}, nil
	}
	return nil, errors.New("not implemented")
}

func (s *stubUserRepository) GetByID(_ context.Context, id uint) (*model.User, error) {
	if s.userByDomain != nil && s.userByDomain.ID == id {
		return s.userByDomain, nil
	}
	if user, ok := s.userByID[id]; ok {
		return user, nil
	}
	return nil, repository.ErrUserNotExists
}

func (s *stubUserRepository) GetByEmail(_ context.Context, email string) (*model.User, error) {
	if user, ok := s.userByEmail[email]; ok {
		return user, nil
	}
	return nil, repository.ErrUserNotExists
}

func (s *stubUserRepository) GetByDomain(_ context.Context, domain string) (*model.User, error) {
	if s.userByDomain != nil && s.userByDomain.Domain == domain {
		return s.userByDomain, nil
	}
	return nil, repository.ErrUserNotExists
}

func (s *stubUserRepository) Update(context.Context, uint, repository.UpdateUserOptions) error {
	return errors.New("not implemented")
}

func (s *stubUserRepository) UpdateRoutingWorkspace(_ context.Context, id, workspaceID uint) (*model.User, error) {
	user, err := s.GetByID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	user.RoutingWorkspaceID = workspaceID
	return user, nil
}

func (s *stubUserRepository) UpdateHarassmentSetting(context.Context, uint, repository.HarassmentSettingOptions) error {
	return errors.New("not implemented")
}

func (s *stubUserRepository) Authenticate(context.Context, string, string) (*model.User, error) {
	return nil, errors.New("not implemented")
}

func (s *stubUserRepository) ChangePassword(context.Context, uint, string, string) error {
	return errors.New("not implemented")
}

func (s *stubUserRepository) UpdatePassword(context.Context, uint, string) error {
	return errors.New("not implemented")
}

func (s *stubUserRepository) Deactivate(context.Context, uint) error {
	return errors.New("not implemented")
}

type stubQuestionRepository struct {
	createCalls            []repository.CreateQuestionOptions
	questionsByID          map[uint]*model.Question
	questionsByWorkspaceID map[uint][]*model.Question
	workspaceStats         *repository.WorkspaceQuestionStats
	totalCount             int64
	answeredCount          int64
	unreadCount            int64
	getByIDErr             error
	unreadCountErr         error
	markViewedErr          error
	markAllViewedErr       error
}

func (s *stubQuestionRepository) Create(_ context.Context, opts repository.CreateQuestionOptions) (*model.Question, error) {
	s.createCalls = append(s.createCalls, opts)

	question := &model.Question{
		UserID:            opts.UserID,
		TenantID:          opts.TenantID,
		WorkspaceID:       opts.WorkspaceID,
		Content:           opts.Content,
		ReceiveReplyEmail: opts.ReceiveReplyEmail,
		AskerUserID:       opts.AskerUserID,
		Status:            model.QuestionStatusNew,
		IsPrivate:         opts.IsPrivate,
	}
	if s.questionsByID != nil {
		id := uint(len(s.questionsByID) + 1)
		question.ID = id
		s.questionsByID[id] = cloneQuestion(question)
	}

	return question, nil
}

func (s *stubQuestionRepository) GetByID(_ context.Context, id uint) (*model.Question, error) {
	if s.getByIDErr != nil {
		return nil, s.getByIDErr
	}
	question, ok := s.questionsByID[id]
	if !ok {
		return nil, repository.ErrQuestionNotExist
	}
	return cloneQuestion(question), nil
}

func (s *stubQuestionRepository) GetByUserID(context.Context, uint, repository.GetQuestionsByUserIDOptions) ([]*model.Question, error) {
	return nil, errors.New("not implemented")
}

func (s *stubQuestionRepository) GetByWorkspaceID(_ context.Context, workspaceID uint, _ repository.GetQuestionsByWorkspaceIDOptions) ([]*model.Question, error) {
	questions := s.questionsByWorkspaceID[workspaceID]
	clones := make([]*model.Question, 0, len(questions))
	for _, question := range questions {
		clones = append(clones, cloneQuestion(question))
	}
	return clones, nil
}

func (s *stubQuestionRepository) GetByAskUserID(context.Context, uint, repository.GetQuestionsByAskUserIDOptions) ([]*model.Question, error) {
	return nil, errors.New("not implemented")
}

func (s *stubQuestionRepository) AnswerByID(_ context.Context, id uint, answer string) error {
	question, ok := s.questionsByID[id]
	if !ok {
		return repository.ErrQuestionNotExist
	}
	question.Answer = answer
	if strings.TrimSpace(answer) == "" {
		question.Status = model.QuestionStatusInProgress
		question.ResolvedAt = nil
	} else {
		question.Status = model.QuestionStatusAnswered
		now := time.Now()
		question.ResolvedAt = &now
	}
	return nil
}

func (s *stubQuestionRepository) UpdateStatus(_ context.Context, id uint, status model.QuestionStatus) (*model.Question, error) {
	question, ok := s.questionsByID[id]
	if !ok {
		return nil, repository.ErrQuestionNotExist
	}
	question.Status = status
	if status.IsResolved() {
		now := time.Now()
		question.ResolvedAt = &now
	} else {
		question.ResolvedAt = nil
	}
	return cloneQuestion(question), nil
}

func (s *stubQuestionRepository) UpdateAssignment(_ context.Context, id uint, assignedToUserID *uint) (*model.Question, error) {
	question, ok := s.questionsByID[id]
	if !ok {
		return nil, repository.ErrQuestionNotExist
	}
	question.AssignedToUserID = assignedToUserID
	return cloneQuestion(question), nil
}

func (s *stubQuestionRepository) UpdateInternalNote(_ context.Context, id uint, internalNote string) (*model.Question, error) {
	question, ok := s.questionsByID[id]
	if !ok {
		return nil, repository.ErrQuestionNotExist
	}
	question.InternalNote = strings.TrimSpace(internalNote)
	return cloneQuestion(question), nil
}

func (s *stubQuestionRepository) MarkViewed(_ context.Context, id uint, viewedAt time.Time) error {
	if s.markViewedErr != nil {
		return s.markViewedErr
	}
	question, ok := s.questionsByID[id]
	if !ok {
		return repository.ErrQuestionNotExist
	}
	question.ViewedAt = &viewedAt
	return nil
}

func (s *stubQuestionRepository) MarkAllViewed(_ context.Context, userID uint, viewedAt time.Time) (int64, error) {
	if s.markAllViewedErr != nil {
		return 0, s.markAllViewedErr
	}
	var marked int64
	for _, question := range s.questionsByID {
		if question.UserID != userID || question.ViewedAt != nil {
			continue
		}
		question.ViewedAt = &viewedAt
		marked++
	}
	return marked, nil
}

func (s *stubQuestionRepository) DeleteByID(_ context.Context, id uint) error {
	if _, ok := s.questionsByID[id]; !ok {
		return repository.ErrQuestionNotExist
	}
	delete(s.questionsByID, id)
	for workspaceID, questions := range s.questionsByWorkspaceID {
		filtered := make([]*model.Question, 0, len(questions))
		for _, question := range questions {
			if question.ID != id {
				filtered = append(filtered, question)
			}
		}
		s.questionsByWorkspaceID[workspaceID] = filtered
	}
	return nil
}

func (s *stubQuestionRepository) Count(_ context.Context, _ uint, opts repository.GetQuestionsCountOptions) (int64, error) {
	if opts.FilterAnswered {
		return s.answeredCount, nil
	}
	return s.totalCount, nil
}

func (s *stubQuestionRepository) CountUnread(context.Context, uint, bool) (int64, error) {
	if s.unreadCountErr != nil {
		return 0, s.unreadCountErr
	}
	return s.unreadCount, nil
}

func (s *stubQuestionRepository) GetWorkspaceStats(context.Context, uint, repository.GetWorkspaceQuestionStatsOptions) (*repository.WorkspaceQuestionStats, error) {
	if s.workspaceStats == nil {
		return &repository.WorkspaceQuestionStats{}, nil
	}
	cloned := *s.workspaceStats
	return &cloned, nil
}

func (s *stubQuestionRepository) SetPrivate(_ context.Context, id uint) error {
	question, ok := s.questionsByID[id]
	if !ok {
		return repository.ErrQuestionNotExist
	}
	question.IsPrivate = true
	return nil
}

func (s *stubQuestionRepository) SetPublic(_ context.Context, id uint) error {
	question, ok := s.questionsByID[id]
	if !ok {
		return repository.ErrQuestionNotExist
	}
	question.IsPrivate = false
	return nil
}

func cloneQuestion(question *model.Question) *model.Question {
	if question == nil {
		return nil
	}
	cloned := *question
	return &cloned
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

type stubReplyEmailSender struct {
	calls []notify.ReplyEmailInput
	err   error
}

func (s *stubReplyEmailSender) SendQuestionAnswered(_ context.Context, input notify.ReplyEmailInput) error {
	s.calls = append(s.calls, input)
	return s.err
}

func testAuthMiddleware(user *model.User) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := appcontext.FromGin(c)
		ctx.User = user
		ctx.IsLogged = user != nil
		c.Next()
	}
}

func requireAPIError(t *testing.T, recorder *httptest.ResponseRecorder, statusCode, errorCode int, message string) {
	t.Helper()

	require.Equal(t, statusCode, recorder.Code)

	var response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, errorCode, response.Code)
	require.Equal(t, message, response.Message)
}
