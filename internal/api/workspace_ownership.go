package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"qnetward/internal/store"
)

// ownershipRequest names who becomes the owner. Confirm must repeat the
// workspace's name, so a slip of the mouse can't give a workspace away.
type ownershipRequest struct {
	UserID  int64  `json:"user_id"`
	Confirm string `json:"confirm"`
}

// teamActor is the signed-in user and their active workspace, for the team
// endpoints that change membership.
func (srv *Server) teamActor(w http.ResponseWriter, r *http.Request) (store.User, store.WorkspaceContext, bool) {
	if srv.store == nil {
		writeJSON(w, http.StatusServiceUnavailable, apiError{Error: "storage not available"})
		return store.User{}, store.WorkspaceContext{}, false
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, apiError{Error: "method not allowed"})
		return store.User{}, store.WorkspaceContext{}, false
	}
	if !srv.isAllowedRequestOrigin(r) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "forbidden origin"})
		return store.User{}, store.WorkspaceContext{}, false
	}
	user, ok := srv.requireAuthenticatedUser(w, r)
	if !ok {
		return store.User{}, store.WorkspaceContext{}, false
	}
	if srv.isDemoUser(user) {
		writeJSON(w, http.StatusForbidden, apiError{Error: "the shared demo account can't change its team"})
		return store.User{}, store.WorkspaceContext{}, false
	}
	_, _, activeWorkspaceID, _ := srv.currentSessionState(r)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	workspace, err := srv.resolveActiveWorkspace(ctx, user, activeWorkspaceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, apiError{Error: err.Error()})
		return store.User{}, store.WorkspaceContext{}, false
	}
	return user, workspace, true
}

// handleTransferOwnership makes another member the owner; the current owner
// becomes an admin.
func (srv *Server) handleTransferOwnership(w http.ResponseWriter, r *http.Request) {
	user, workspace, ok := srv.teamActor(w, r)
	if !ok {
		return
	}
	var req ownershipRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, apiError{Error: err.Error()})
		return
	}
	if !strings.EqualFold(strings.TrimSpace(req.Confirm), strings.TrimSpace(workspace.Name)) {
		writeJSON(w, http.StatusBadRequest, apiError{Error: "type the workspace's name to confirm"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	member, err := srv.store.TransferWorkspaceOwnership(ctx, workspace.ID, user.ID, req.UserID)
	switch {
	case errors.Is(err, store.ErrWorkspaceMemberNotFound):
		writeJSON(w, http.StatusNotFound, apiError{Error: "that person isn't a member of this workspace"})
		return
	case errors.Is(err, store.ErrWorkspacePermissionDenied):
		writeJSON(w, http.StatusForbidden, apiError{Error: "only an owner can hand over ownership, to someone else in the workspace"})
		return
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "couldn't transfer ownership"})
		return
	}
	srv.recordAudit(r, workspace.ID, user, "member.ownership_transferred", idTarget("user", member.UserID, member.Email), nil)
	writeJSON(w, http.StatusOK, authMessageResponse{OK: true, Message: fmt.Sprintf("%s now owns %s. You're an admin.", member.Email, workspace.Name), Email: member.Email})
}

// handleLeaveWorkspace takes the signed-in user out of their active workspace.
// The console then switches to another of their workspaces.
func (srv *Server) handleLeaveWorkspace(w http.ResponseWriter, r *http.Request) {
	user, workspace, ok := srv.teamActor(w, r)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	err := srv.store.LeaveWorkspace(ctx, workspace.ID, user.ID)
	switch {
	case errors.Is(err, store.ErrWorkspaceLastOwner):
		writeJSON(w, http.StatusConflict, apiError{Error: "you're the only owner; transfer ownership to someone else before leaving"})
		return
	case errors.Is(err, store.ErrWorkspaceMemberNotFound):
		writeJSON(w, http.StatusNotFound, apiError{Error: "you're not a member of this workspace"})
		return
	case err != nil:
		writeJSON(w, http.StatusInternalServerError, apiError{Error: "couldn't leave the workspace"})
		return
	}
	srv.recordAudit(r, workspace.ID, user, "member.left", idTarget("user", user.ID, user.Email), nil)
	writeJSON(w, http.StatusOK, authMessageResponse{OK: true, Message: fmt.Sprintf("You left %s.", workspace.Name)})
}
