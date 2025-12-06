package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/S0rgi/Gainly_Avatars/internal/middleware"
	"github.com/S0rgi/Gainly_Avatars/internal/services"
)

type Handlers struct {
	avatarService *services.AvatarService
}

func NewHandlers(avatarService *services.AvatarService) *Handlers {
	return &Handlers{
		avatarService: avatarService,
	}
}

// AddAvatar обрабатывает загрузку аватарки
// @Summary Загрузить аватарку
// @Description Загружает новую аватарку для текущего пользователя
// @Tags avatars
// @Accept multipart/form-data
// @Produce json
// @Param avatar formData file true "Файл аватарки"
// @Success 200 {object} map[string]string "GUID загруженной аватарки"
// @Failure 400 {object} map[string]string "Ошибка валидации"
// @Failure 401 {object} map[string]string "Не авторизован"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/avatar [post]
func (h *Handlers) AddAvatar(w http.ResponseWriter, r *http.Request) {
	// Получаем пользователя из контекста
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "User not found in context", http.StatusInternalServerError)
		return
	}

	// Парсим multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB max
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to parse multipart form")
		return
	}

	// Получаем файл
	file, handler, err := r.FormFile("avatar")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to get file from form")
		return
	}
	defer file.Close()

	// Получаем размер файла
	fileSize := handler.Size

	// Получаем content type
	contentType := handler.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Загружаем аватарку
	guid, err := h.avatarService.AddAvatar(
		r.Context(),
		user.Username,
		file,
		handler.Filename,
		contentType,
		fileSize,
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Возвращаем GUID
	respondWithJSON(w, http.StatusOK, map[string]string{
		"guid": guid,
	})
}

// GetAvatarsByUsernames обрабатывает получение аватарок по списку username
// @Summary Получить аватарки по username
// @Description Возвращает URL аватарок для списка пользователей
// @Tags avatars
// @Accept json
// @Produce json
// @Param request body GetAvatarsRequest true "Список username"
// @Success 200 {object} map[string]string "Карта username -> URL"
// @Failure 400 {object} map[string]string "Ошибка валидации"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /api/avatars [post]
func (h *Handlers) GetAvatarsByUsernames(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Usernames []string `json:"usernames"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(request.Usernames) == 0 {
		respondWithError(w, http.StatusBadRequest, "Usernames list cannot be empty")
		return
	}

	// Получаем аватарки
	avatars, err := h.avatarService.GetAvatarsByUsernames(r.Context(), request.Usernames)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, avatars)
}

// GetMyAvatar обрабатывает получение своей аватарки
// @Summary Получить свою аватарку
// @Description Возвращает URL аватарки текущего пользователя
// @Tags avatars
// @Produce json
// @Success 200 {object} map[string]string "URL аватарки"
// @Failure 401 {object} map[string]string "Не авторизован"
// @Failure 404 {object} map[string]string "Аватарка не найдена"
// @Security BearerAuth
// @Router /api/avatar/me [get]
func (h *Handlers) GetMyAvatar(w http.ResponseWriter, r *http.Request) {
	// Получаем пользователя из контекста
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusInternalServerError, "User not found in context")
		return
	}

	// Получаем аватарку
	url, err := h.avatarService.GetMyAvatar(r.Context(), user.Username)
	if err != nil {
		respondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"url": url,
	})
}

// DeleteMyAvatar обрабатывает удаление своей аватарки
// @Summary Удалить свою аватарку
// @Description Удаляет аватарку текущего пользователя
// @Tags avatars
// @Success 204 "Аватарка успешно удалена"
// @Failure 401 {object} map[string]string "Не авторизован"
// @Failure 404 {object} map[string]string "Аватарка не найдена"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /api/avatar/me [delete]
func (h *Handlers) DeleteMyAvatar(w http.ResponseWriter, r *http.Request) {
	// Получаем пользователя из контекста
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusInternalServerError, "User not found in context")
		return
	}

	// Удаляем аватарку
	err := h.avatarService.DeleteMyAvatar(r.Context(), user.Username)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Вспомогательные функции для ответов

type GetAvatarsRequest struct {
	Usernames []string `json:"usernames" example:"user1,user2"`
}

type ErrorResponse struct {
	Error string `json:"error" example:"error message"`
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
