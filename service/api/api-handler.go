package api

import (
	"net/http"
)

// Handler returns an instance of httprouter.Router that handle APIs registered here
func (rt *_router) Handler() http.Handler {
	// Register routes
	rt.router.GET("/", rt.getHelloWorld)
	rt.router.GET("/context", rt.wrap(rt.getContextReply))

	// WASAText routes
	rt.router.POST("/session", rt.wrap(rt.doLogin))
	rt.router.GET("/users/me", rt.wrap(rt.getMyProfile))
	rt.router.GET("/users", rt.wrap(rt.listUsers))
	rt.router.PUT("/users/me/name", rt.wrap(rt.setMyUserName))
	rt.router.PUT("/users/me/photo", rt.wrap(rt.setMyPhoto))

	rt.router.POST("/conversations", rt.wrap(rt.createConversation))
	rt.router.GET("/conversations", rt.wrap(rt.getMyConversations))
	rt.router.GET("/conversations/:conversationId", rt.wrap(rt.getConversation))
	rt.router.POST("/groups/:groupId/members", rt.wrap(rt.addToGroup))
	rt.router.DELETE("/groups/:groupId/members/me", rt.wrap(rt.leaveGroup))
	rt.router.PUT("/groups/:groupId/name", rt.wrap(rt.setGroupName))
	rt.router.PUT("/groups/:groupId/photo", rt.wrap(rt.setGroupPhoto))

	rt.router.POST("/conversations/:conversationId/messages", rt.wrap(rt.sendMessage))
	rt.router.PUT("/messages/:messageId/read", rt.wrap(rt.markMessageAsRead))
	rt.router.DELETE("/messages/:messageId", rt.wrap(rt.deleteMessage))
	rt.router.POST("/messages/:messageId/forward", rt.wrap(rt.forwardMessage))

	rt.router.POST("/messages/:messageId/comments", rt.wrap(rt.commentMessage))
	rt.router.DELETE("/messages/:messageId/comments/:commentId", rt.wrap(rt.uncommentMessage))

	// Special routes
	rt.router.GET("/liveness", rt.liveness)

	return rt.router
}
