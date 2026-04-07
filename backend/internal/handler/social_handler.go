package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/shout/ai-study-tool/backend/internal/domain"
	"github.com/shout/ai-study-tool/backend/internal/handler/dto"
	"github.com/shout/ai-study-tool/backend/internal/usecase"
)

type SocialHandler struct {
	socialUsecase *usecase.SocialUsecase
}

func NewSocialHandler(socialUsecase *usecase.SocialUsecase) *SocialHandler {
	return &SocialHandler{socialUsecase: socialUsecase}
}

func (h *SocialHandler) Follow(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	targetID := r.PathValue("id")
	if targetID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "user id is required")
		return
	}

	if err := h.socialUsecase.Follow(r.Context(), userID, targetID); err != nil {
		h.writeSocialError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "followed"})
}

func (h *SocialHandler) Unfollow(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	targetID := r.PathValue("id")
	if targetID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "user id is required")
		return
	}

	if err := h.socialUsecase.Unfollow(r.Context(), userID, targetID); err != nil {
		h.writeSocialError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "unfollowed"})
}

func (h *SocialHandler) Like(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	postID := r.PathValue("id")
	if postID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "post id is required")
		return
	}

	if err := h.socialUsecase.Like(r.Context(), userID, postID); err != nil {
		h.writeSocialError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "liked"})
}

func (h *SocialHandler) Unlike(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	postID := r.PathValue("id")
	if postID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "post id is required")
		return
	}

	if err := h.socialUsecase.Unlike(r.Context(), userID, postID); err != nil {
		h.writeSocialError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "unliked"})
}

func (h *SocialHandler) Repost(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	postID := r.PathValue("id")
	if postID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "post id is required")
		return
	}

	if err := h.socialUsecase.Repost(r.Context(), userID, postID); err != nil {
		h.writeSocialError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "reposted"})
}

func (h *SocialHandler) Unrepost(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	postID := r.PathValue("id")
	if postID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "post id is required")
		return
	}

	if err := h.socialUsecase.Unrepost(r.Context(), userID, postID); err != nil {
		h.writeSocialError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "unreposted"})
}

func (h *SocialHandler) CreateComment(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}

	postID := r.PathValue("id")
	if postID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "post id is required")
		return
	}

	var req dto.CreateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	comment, err := h.socialUsecase.CreateComment(r.Context(), &domain.Comment{
		PostID:  postID,
		UserID:  userID,
		Content: req.Content,
	})
	if err != nil {
		h.writeSocialError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toCommentResponse(comment))
}

func (h *SocialHandler) ListComments(w http.ResponseWriter, r *http.Request) {
	postID := r.PathValue("id")
	if postID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "post id is required")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	comments, err := h.socialUsecase.ListComments(r.Context(), postID, limit, offset)
	if err != nil {
		h.writeSocialError(w, err)
		return
	}

	responses := make([]dto.CommentResponse, 0, len(comments))
	for _, comment := range comments {
		responses = append(responses, toCommentResponse(comment))
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"comments": responses,
		"limit":    limit,
		"offset":   offset,
	})
}

func (h *SocialHandler) writeSocialError(w http.ResponseWriter, err error) {
	if strings.HasPrefix(err.Error(), "validation:") {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", strings.TrimPrefix(err.Error(), "validation: "))
		return
	}

	writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
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
