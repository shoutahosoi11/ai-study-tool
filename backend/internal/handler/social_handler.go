package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/handler/dto"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type SocialHandler struct {
	socialUsecase *usecase.SocialUsecase
	userUsecase   *usecase.UserUsecase
}

func NewSocialHandler(socialUsecase *usecase.SocialUsecase, userUsecase *usecase.UserUsecase) *SocialHandler {
	return &SocialHandler{socialUsecase: socialUsecase, userUsecase: userUsecase}
}

func (h *SocialHandler) currentUserID(c echo.Context) (string, error) {
	firebaseUID, ok := c.Get("firebase_uid").(string)
	if !ok || firebaseUID == "" {
		return "", echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	user, err := h.userUsecase.GetByFirebaseUID(c.Request().Context(), firebaseUID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return "", echo.NewHTTPError(http.StatusNotFound, "user not found")
		}
		return "", echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
	}
	return user.ID.String(), nil
}

func (h *SocialHandler) Follow(c echo.Context) error {
	userID, err := h.currentUserID(c)
	if err != nil {
		return err
	}
	targetID := c.Param("id")
	if targetID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "user id is required")
	}
	if err := h.socialUsecase.Follow(c.Request().Context(), userID, targetID); err != nil {
		return h.socialError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "followed"})
}

func (h *SocialHandler) Unfollow(c echo.Context) error {
	userID, err := h.currentUserID(c)
	if err != nil {
		return err
	}
	targetID := c.Param("id")
	if targetID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "user id is required")
	}
	if err := h.socialUsecase.Unfollow(c.Request().Context(), userID, targetID); err != nil {
		return h.socialError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "unfollowed"})
}

func (h *SocialHandler) Like(c echo.Context) error {
	userID, err := h.currentUserID(c)
	if err != nil {
		return err
	}
	postID := c.Param("id")
	if postID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "post id is required")
	}
	if err := h.socialUsecase.Like(c.Request().Context(), userID, postID); err != nil {
		return h.socialError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "liked"})
}

func (h *SocialHandler) Unlike(c echo.Context) error {
	userID, err := h.currentUserID(c)
	if err != nil {
		return err
	}
	postID := c.Param("id")
	if postID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "post id is required")
	}
	if err := h.socialUsecase.Unlike(c.Request().Context(), userID, postID); err != nil {
		return h.socialError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "unliked"})
}

func (h *SocialHandler) Repost(c echo.Context) error {
	userID, err := h.currentUserID(c)
	if err != nil {
		return err
	}
	postID := c.Param("id")
	if postID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "post id is required")
	}
	if err := h.socialUsecase.Repost(c.Request().Context(), userID, postID); err != nil {
		return h.socialError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "reposted"})
}

func (h *SocialHandler) Unrepost(c echo.Context) error {
	userID, err := h.currentUserID(c)
	if err != nil {
		return err
	}
	postID := c.Param("id")
	if postID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "post id is required")
	}
	if err := h.socialUsecase.Unrepost(c.Request().Context(), userID, postID); err != nil {
		return h.socialError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "unreposted"})
}

func (h *SocialHandler) CreateComment(c echo.Context) error {
	userID, err := h.currentUserID(c)
	if err != nil {
		return err
	}
	postID := c.Param("id")
	if postID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "post id is required")
	}

	req := new(dto.CreateCommentRequest)
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	comment, err := h.socialUsecase.CreateComment(c.Request().Context(), &domain.Comment{
		PostID:  postID,
		UserID:  userID,
		Content: req.Content,
	})
	if err != nil {
		return h.socialError(err)
	}

	return c.JSON(http.StatusCreated, toCommentResponse(comment))
}

func (h *SocialHandler) ListComments(c echo.Context) error {
	postID := c.Param("id")
	if postID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "post id is required")
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))

	comments, err := h.socialUsecase.ListComments(c.Request().Context(), postID, limit, offset)
	if err != nil {
		return h.socialError(err)
	}

	responses := make([]dto.CommentResponse, 0, len(comments))
	for _, comment := range comments {
		responses = append(responses, toCommentResponse(comment))
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"comments": responses,
		"limit":    limit,
		"offset":   offset,
	})
}

func (h *SocialHandler) socialError(err error) error {
	if errors.Is(err, domain.ErrNotFound) {
		return echo.NewHTTPError(http.StatusNotFound, "not found")
	}
	if strings.HasPrefix(err.Error(), "validation:") {
		return echo.NewHTTPError(http.StatusBadRequest, strings.TrimPrefix(err.Error(), "validation: "))
	}
	return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
}

func toCommentResponse(comment *domain.Comment) dto.CommentResponse {
	return dto.CommentResponse{
		ID:        comment.ID,
		PostID:    comment.PostID,
		UserID:    comment.UserID,
		Content:   comment.Content,
		CreatedAt: comment.CreatedAt,
	}
}
