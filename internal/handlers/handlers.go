package handlers

import (
	"bytes"
	"encoding/json"
	"io"
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
// @Router /avatar [post]
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

// GetAvatar обрабатывает получение аватарки по username
// @Summary Получить аватарку по username
// @Description Возвращает URL аватарки указанного пользователя
// @Tags avatars
// @Produce json
// @Param username query string true "Имя пользователя"
// @Success 200 {object} map[string]string "URL аватарки"
// @Failure 400 {object} map[string]string "Ошибка валидации"
// @Failure 404 {object} map[string]string "Аватарка не найдена"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Security BearerAuth
// @Router /avatar [get]
func (h *Handlers) GetAvatar(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")

	if username == "" {
		respondWithError(w, http.StatusBadRequest, "username is required")
		return
	}

	url, err := h.avatarService.GetMyAvatar(r.Context(), username)
	if err != nil {
		respondWithError(w, http.StatusNotFound, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"url": url,
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
// @Router /avatars [post]
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
// @Router /avatar/me [get]
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
// @Router /avatar/me [delete]
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

// UploadAvatarFromURL загружает аватарку по URL (например Telegram)
// @Summary Загрузить аватарку по URL
// @Description Загружает аватарку из внешнего URL (например Telegram File API)
// @Tags avatars
// @Accept json
// @Produce json
// @Param request body UploadAvatarFromURLRequest true "URL изображения"
// @Success 200 {object} map[string]string "GUID загруженной аватарки"
// @Failure 400 {object} map[string]string "Ошибка валидации"
// @Failure 401 {object} map[string]string "Не авторизован"
// @Failure 500 {object} map[string]string "Ошибка загрузки или хранения"
// @Security BearerAuth
// @Router /avatar/url [post]
func (h *Handlers) UploadAvatarFromURL(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		respondWithError(w, http.StatusUnauthorized, "User not found in context")
		return
	}

	var req UploadAvatarFromURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if req.URL == "" {
		respondWithError(w, http.StatusBadRequest, "url is required")
		return
	}

	// Загружаем файл
	resp, err := http.Get(req.URL)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Failed to download file")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respondWithError(w, http.StatusBadRequest, "Remote server returned error")
		return
	}

	// Определяем content-type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Определяем длину (если Telegram не даёт — читаем вручную)
	contentLength := resp.ContentLength
	if contentLength <= 0 {
		// Чтение в память для получения размера
		fileData, err := io.ReadAll(resp.Body)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to read file")
			return
		}

		contentLength = int64(len(fileData))
		fileReader := io.NopCloser(bytes.NewReader(fileData))

		guid, err := h.avatarService.AddAvatar(
			r.Context(),
			user.Username,
			fileReader,
			"avatar.jpg", // или req.URL basename?
			contentType,
			contentLength,
		)

		if err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondWithJSON(w, http.StatusOK, map[string]string{"guid": guid})
		return
	}

	// Если Content-Length есть — передаем поток напрямую
	guid, err := h.avatarService.AddAvatar(
		r.Context(),
		user.Username,
		resp.Body,
		"avatar.jpg", // filename можно извлечь из URL
		contentType,
		contentLength,
	)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"guid": guid})
}

// Вспомогательные функции для ответов

type UploadAvatarFromURLRequest struct {
	URL string `json:"url" example:"https://t.me/i/userpic/..."`
}

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
