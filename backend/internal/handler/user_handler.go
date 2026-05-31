package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/handler/dto"
	"github.com/shout/ai-study-tool/backend/internal/middleware"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

const (
	maxAvatarURLLength = 2048
	maxBioLength       = 300
)

// ユーザーデータに関する処理をするための部品などをまとめたファイル
type UserHandler struct {
	userUsecase usecase.UserUsecaseInterface
}

func NewUserHandler(userUsecase usecase.UserUsecaseInterface) *UserHandler {
	return &UserHandler{userUsecase: userUsecase}
}

func (h *UserHandler) SignUp(c echo.Context) error {
	firebaseUID, ok := middleware.GetFirebaseUID(c)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	// usernameやdisplay_nameなどをリクエストボディから受け取る
	//　DBからログイン中のユーザーのusernameを取得しているわけではない
	req := new(dto.SignUpRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	// usecaseに渡す形に整形
	input, err := buildCreateUserInput(firebaseUID, authEmail(c), req)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	// 既存ユーザー確認や username 重複確認
	user, err := h.userUsecase.SignUp(c.Request().Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid user input")
		}
		if errors.Is(err, domain.ErrAlreadyExists) {
			return echo.NewHTTPError(http.StatusConflict, "username already taken")
		}
		log.Printf("user signup error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.ToMeResponse(user))
}

func (h *UserHandler) GetMe(c echo.Context) error {
	user, err := resolveCurrentUser(c, h.userUsecase, "user")
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, dto.ToMeResponse(user))
}

func (h *UserHandler) GetUser(c echo.Context) error {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid user id")
	}

	user, err := h.userUsecase.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		log.Printf("user get public profile error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.ToPublicUserProfileResponse(user))
}

func (h *UserHandler) UpdateProfile(c echo.Context) error {
	// firebaseUID から me を取得しているのは、「誰のプロフィールを更新するか」を安全に確定
	me, err := resolveCurrentUser(c, h.userUsecase, "user updateProfile")
	if err != nil {
		return err
	}
	req, fields, err := decodeUpdateProfileRequest(c)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	input, err := buildUpdateUserInput(req, fields)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	// me.ID を使っているので、リクエスト側が勝手に他人のIDを指定して更新する形にはなっていません。
	updated, err := h.userUsecase.UpdateProfile(c.Request().Context(), me.ID, input)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid user input")
		}
		if errors.Is(err, domain.ErrAlreadyExists) {
			return echo.NewHTTPError(http.StatusConflict, "username already taken")
		}
		log.Printf("user updateProfile error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.ToMeResponse(updated))
}

func (h *UserHandler) UpdateQuestionSettings(c echo.Context) error {
	me, err := resolveCurrentUser(c, h.userUsecase, "user updateQuestionSettings")
	if err != nil {
		return err
	}

	req := new(dto.UpdateQuestionSettingsRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if !domain.IsValidDefaultQuestionCount(req.DefaultQuestionCount) {
		return echo.NewHTTPError(http.StatusBadRequest, "default_question_count must be 0 or between 1 and 10")
	}

	updated, err := h.userUsecase.UpdateQuestionSettings(c.Request().Context(), me.ID, domain.UpdateQuestionSettingsInput{
		DefaultQuestionCount: req.DefaultQuestionCount,
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid question settings")
		}
		log.Printf("user updateQuestionSettings error: %v", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}

	return c.JSON(http.StatusOK, dto.ToMeResponse(updated))
}

func buildCreateUserInput(firebaseUID string, email *string, req *dto.SignUpRequest) (domain.CreateUserInput, error) {
	firebaseUID = strings.TrimSpace(firebaseUID)
	email = normalizeOptionalText(email)
	if email != nil {
		normalizedEmail := strings.ToLower(*email)
		email = &normalizedEmail
	}

	username := domain.NormalizeUsername(req.Username)
	if err := validateUsername(username); err != nil {
		return domain.CreateUserInput{}, err
	}

	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = username
	}
	if err := validateRequiredText(displayName, "display_name", 1, 100); err != nil {
		return domain.CreateUserInput{}, err
	}
	university := normalizeOptionalText(req.University)
	if err := validateOptionalText(university, "university", 100); err != nil {
		return domain.CreateUserInput{}, err
	}
	faculty := normalizeOptionalText(req.Faculty)
	if err := validateOptionalText(faculty, "faculty", 100); err != nil {
		return domain.CreateUserInput{}, err
	}
	country := normalizeOptionalText(req.Country)
	if err := validateOptionalText(country, "country", 10); err != nil {
		return domain.CreateUserInput{}, err
	}
	avatarURL := normalizeOptionalText(req.AvatarURL)
	if err := validateOptionalURL(avatarURL, "avatar_url", maxAvatarURLLength); err != nil {
		return domain.CreateUserInput{}, err
	}

	return domain.CreateUserInput{
		FirebaseUID: firebaseUID,
		Email:       email,
		Username:    username,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
		University:  university,
		Faculty:     faculty,
		Grade:       req.Grade,
		Country:     country,
	}, nil
}

func authEmail(c echo.Context) *string {
	claims, ok := middleware.GetAuthClaims(c)
	if !ok {
		return nil
	}
	if verified, ok := claims["email_verified"].(bool); !ok || !verified {
		return nil
	}
	email, ok := claims["email"].(string)
	if !ok {
		return nil
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil
	}
	return &email
}

func decodeUpdateProfileRequest(c echo.Context) (*dto.UpdateProfileRequest, map[string]bool, error) {
	var raw map[string]json.RawMessage
	decoder := json.NewDecoder(c.Request().Body)
	if err := decoder.Decode(&raw); err != nil {
		return nil, nil, err
	}
	if raw == nil {
		return nil, nil, errors.New("request body must be a JSON object")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, nil, errors.New("request body must contain a single JSON object")
	}

	body, err := json.Marshal(raw)
	if err != nil {
		return nil, nil, err
	}

	req := new(dto.UpdateProfileRequest)
	if err := json.Unmarshal(body, req); err != nil {
		return nil, nil, err
	}

	fields := make(map[string]bool, len(raw))
	for key := range raw {
		fields[key] = true
	}
	return req, fields, nil
}

func buildUpdateUserInput(req *dto.UpdateProfileRequest, fields map[string]bool) (domain.UpdateUserInput, error) {
	var input domain.UpdateUserInput
	if fields["username"] {
		username := domain.NormalizeUsername(req.Username)
		if err := validateUsername(username); err != nil {
			return domain.UpdateUserInput{}, err
		}
		input.Username = domain.SomeStringUpdate(username)
	}
	if fields["display_name"] {
		displayName := strings.TrimSpace(req.DisplayName)
		if err := validateRequiredText(displayName, "display_name", 1, 100); err != nil {
			return domain.UpdateUserInput{}, err
		}
		input.DisplayName = domain.SomeStringUpdate(displayName)
	}
	if fields["avatar_url"] {
		avatarURL := normalizeOptionalText(req.AvatarURL)
		if err := validateOptionalURL(avatarURL, "avatar_url", maxAvatarURLLength); err != nil {
			return domain.UpdateUserInput{}, err
		}
		input.AvatarURL = optionalStringUpdate(avatarURL)
	}
	if fields["bio"] {
		bio := normalizeOptionalText(req.Bio)
		if err := validateOptionalText(bio, "bio", maxBioLength); err != nil {
			return domain.UpdateUserInput{}, err
		}
		input.Bio = optionalStringUpdate(bio)
	}
	if fields["university"] {
		university := normalizeOptionalText(req.University)
		if err := validateOptionalText(university, "university", 100); err != nil {
			return domain.UpdateUserInput{}, err
		}
		input.University = optionalStringUpdate(university)
	}
	if fields["faculty"] {
		faculty := normalizeOptionalText(req.Faculty)
		if err := validateOptionalText(faculty, "faculty", 100); err != nil {
			return domain.UpdateUserInput{}, err
		}
		input.Faculty = optionalStringUpdate(faculty)
	}
	if fields["grade"] {
		input.Grade = optionalInt16Update(req.Grade)
	}
	if fields["country"] {
		country := normalizeOptionalText(req.Country)
		if err := validateOptionalText(country, "country", 10); err != nil {
			return domain.UpdateUserInput{}, err
		}
		input.Country = optionalStringUpdate(country)
	}
	if !input.HasChanges() {
		return domain.UpdateUserInput{}, errors.New("at least one profile field is required")
	}
	return input, nil
}

func optionalStringUpdate(value *string) domain.OptionalStringUpdate {
	if value == nil {
		return domain.NullStringUpdate()
	}
	return domain.SomeStringUpdate(*value)
}

func optionalInt16Update(value *int16) domain.OptionalInt16Update {
	if value == nil {
		return domain.NullInt16Update()
	}
	return domain.SomeInt16Update(*value)
}

func validateRequiredText(value, field string, minLen, maxLen int) error {
	if value == "" {
		return errors.New(field + " is required")
	}

	length := utf8.RuneCountInString(value)
	if length < minLen {
		return errors.New(field + " must be at least " + strconv.Itoa(minLen) + " characters")
	}
	if length > maxLen {
		return errors.New(field + " must be at most " + strconv.Itoa(maxLen) + " characters")
	}

	return nil
}

func validateUsername(username string) error {
	if username == "" {
		return errors.New("username is required")
	}
	if utf8.RuneCountInString(username) < domain.UsernameMinLength {
		return errors.New("username must be at least " + strconv.Itoa(domain.UsernameMinLength) + " characters")
	}
	if utf8.RuneCountInString(username) > domain.UsernameMaxLength {
		return errors.New("username must be at most " + strconv.Itoa(domain.UsernameMaxLength) + " characters")
	}
	if !domain.IsValidUsername(username) {
		return errors.New("username may contain only lowercase letters, numbers, and underscores")
	}
	return nil
}

func validateOptionalText(value *string, field string, maxLen int) error {
	if value == nil {
		return nil
	}

	if utf8.RuneCountInString(strings.TrimSpace(*value)) > maxLen {
		return errors.New(field + " must be at most " + strconv.Itoa(maxLen) + " characters")
	}

	return nil
}

func validateOptionalURL(value *string, field string, maxLen int) error {
	if value == nil {
		return nil
	}
	if err := validateOptionalText(value, field, maxLen); err != nil {
		return err
	}

	parsed, err := url.ParseRequestURI(*value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return errors.New(field + " must be a valid http or https URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New(field + " must be a valid http or https URL")
	}

	return nil
}

func normalizeOptionalText(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}
