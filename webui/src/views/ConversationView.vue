
<template>
<!-- First simpe version, temporary -->
  <div class="container py-4">
    <div class="mb-3">
      <router-link to="/conversations" class="btn btn-outline-secondary btn-sm">
        Back
      </router-link>
    </div>

    <ErrorMsg v-if="errorMessage" :msg="errorMessage" />
    <LoadingSpinner v-if="loading" />

    <div v-else-if="conversation">
      <div class="mb-3">
        <h1 class="h3 mb-1">
          {{ getConversationTitle() }}
        </h1>
        <div class="text-muted">
          {{ getConversationSubtitle() }}
        </div>
      </div>
      
      <!-- Group management section -->
      <div v-if="isGroupConversation()" class="card mb-4">
        <div class="card-body">
          <h2 class="h5 mb-3">Group Settings</h2>

          <form class="mb-3" @submit.prevent="updateGroupName">
            <label class="form-label">Group name</label>
            <div class="input-group">
              <input
                v-model.trim="groupNameInput"
                type="text"
                class="form-control"
                placeholder="Enter group name"
              />
              <button class="btn btn-outline-primary" type="submit">
                Update Name
              </button>
            </div>
          </form>

          <form @submit.prevent="updateGroupPhoto">
            <label class="form-label">Group photo</label>
            <div class="input-group">
              <input
                v-model.trim="groupPhotoInput"
                type="text"
                class="form-control"
                placeholder="Enter group photo path"
              />
              <button class="btn btn-outline-primary" type="submit">
                Update Photo
              </button>
            </div>
          </form>

          <div class="mb-3 text-muted small">
            Current photo: {{ conversation.photo || "No photo" }}
          </div>

          <hr />
          <div>
            <label class="form-label">Add user to group</label>

            <div v-if="availableUsers.length === 0" class="text-muted small mb-2">
              No more users available to add.
            </div>

            <div v-else class="input-group">
              <select v-model="selectedUserToAdd" class="form-select">
                <option value="">Select a user</option>
                <option
                  v-for="user in availableUsers"
                  :key="user.id"
                  :value="user.id"
                >
                  {{ user.username }}
                </option>
              </select>

              <button class="btn btn-outline-primary" type="button" @click="addUserToGroup">
                Add User
              </button>
            </div>
          </div>
          
          <hr />
          <div class="d-flex justify-content-end">
            <button class="btn btn-outline-danger" type="button" @click="leaveGroup">
              Leave Group
            </button>
          </div>

        </div>
      </div>

      <div class="mb-4">
        <h2 class="h5">Members</h2>
        <ul class="list-group">
          <li
            v-for="member in conversation.members"
            :key="member.id"
            class="list-group-item"
          >
            {{ member.username }}
          </li>
        </ul>
      </div>

      <div class="mb-4">
        <h2 class="h5">Messages</h2>
        <div v-if="conversation.messages.length === 0" class="text-muted">
          No messages yet.
        </div>

        <div
          v-for="message in conversation.messages"
          :key="message.id"
          class="card mb-2"
          :class="isMyMessage(message) ? 'border-primary' : 'border-light'"
        >
          <div class="card-body">
            
            <div class="d-flex justify-content-between align-items-center mb-1">
              <div class="small text-muted">
                {{ isMyMessage(message) ? "You" : message.senderUsername }} • {{ formatDateTime(message.createdAt) }}
              </div>

              <button
                v-if="isMyMessage(message) && !message.deleted"
                class="btn btn-sm btn-outline-danger"
                @click="deleteMessage(message.id)"
              >
                Delete
              </button>
            </div>

            <div :class="message.deleted ? 'text-muted fst-italic' : ''">
              {{ getVisibleMessageContent(message) }}
            </div>

            <div class="mt-2">
              <button
                class="btn btn-sm btn-outline-secondary"
                @click="addComment(message.id, '😀')"
                :disabled="message.deleted"
              >
                😀
              </button>
            </div>

            <div v-if="message.comments && message.comments.length > 0" class="mt-2">
              <div
                v-for="comment in message.comments"
                :key="comment.id"
                class="d-inline-flex align-items-center border rounded-pill px-2 py-1 me-2 mb-2 small bg-light"
              >
                <span class="me-2">
                  {{ comment.content }} {{ comment.authorUsername }}
                </span>

                <button
                  v-if="comment.authorId === currentUserId"
                  class="btn btn-sm btn-link text-danger p-0"
                  @click="deleteComment(message.id, comment.id)"
                  title="Remove reaction"
                >
                  ×
                </button>
              </div>
            </div>

          </div>
        </div>
      </div>

      <form @submit.prevent="sendMessage">
        <div class="input-group">
          <input
            v-model.trim="newMessage"
            type="text"
            class="form-control"
            placeholder="Write a message..."
          />
          <button class="btn btn-primary" type="submit" :disabled="sending">
            Send
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script>
import api from "../services/axios";
import ErrorMsg from "../components/ErrorMsg.vue";
import LoadingSpinner from "../components/LoadingSpinner.vue";

export default {
  name: "ConversationView",
  props: ["conversationId"],
  components: {
    ErrorMsg,
    LoadingSpinner,
  },
  data() {
    return {
      conversation: null,
      loading: false,
      sending: false,
      errorMessage: "",
      newMessage: "",
      currentUserId: localStorage.getItem("token") || "",
      groupNameInput: "",
      groupPhotoInput: "",
      availableUsers: [],
      selectedUserToAdd: "",
    };
  },
  async mounted() {
    await this.loadConversation();
  },
  methods: {
    

    // Returns the text that should be shown for a message.
    // Deleted messages keep their original content in the database, but the UI hides it from the user
    getVisibleMessageContent(message) {
      if (message.deleted) {
        return "[deleted message]";
      }

      return message.content;
    },
    

    async loadConversation() {
      this.loading = true;
      this.errorMessage = "";

      try {
        const response = await api.get(`/conversations/${this.conversationId}`);
        this.conversation = response.data;

        // After loading the conversation, mark all incoming messages as read.
        await this.markMessagesAsRead();

        const refreshedResponse = await api.get(`/conversations/${this.conversationId}`);
        this.conversation = refreshedResponse.data;
        // Keep the editable group fields in sync with the loaded conversation.
        this.groupNameInput = this.conversation.name || "";
        this.groupPhotoInput = this.conversation.photo || "";
        await this.loadAvailableUsersForGroup();
      } catch (e) {
        if (e.response && e.response.data && e.response.data.message) {
          this.errorMessage = e.response.data.message;
        } else {
          this.errorMessage = "Cannot load conversation.";
        }
      } finally {
        this.loading = false;
      }
    },

    // Returns the best title for the current conversation.
    // For groups, use the group name.
    // For direct conversations, use the other user's username.
    getConversationTitle() {
      if (!this.conversation) return "Conversation";

      if (this.conversation.isGroup) {
        return this.conversation.name || "Group";
      }

      const otherMember = (this.conversation.members || []).find(
        (member) => member.id !== this.currentUserId
      );

      return otherMember ? otherMember.username : "Conversation";
    },

    // Returns a small subtitle for the conversation header.
    getConversationSubtitle() {
      if (!this.conversation) return "";

      if (this.conversation.isGroup) {
        const memberCount = (this.conversation.members || []).length;
        return `${memberCount} member${memberCount === 1 ? "" : "s"}`;
      }

      return "Direct conversation";
    },
    
    
    async sendMessage() {
      if (!this.newMessage) return;

      this.sending = true;
      this.errorMessage = "";

      try {
        await api.post(`/conversations/${this.conversationId}/messages`, {
          type: "text",
          content: this.newMessage,
        });

        this.newMessage = "";
        await this.loadConversation();
      } catch (e) {
        if (e.response && e.response.data && e.response.data.message) {
          this.errorMessage = e.response.data.message;
        } else {
          this.errorMessage = "Cannot send message.";
        }
      } finally {
        this.sending = false;
      }
    },
    
    
    // Returns true if the message was sent by the currently logged-in user.
    isMyMessage(message) {
      return message.senderId === this.currentUserId;
    },
    
    
    // Marks all messages from other users as read. Calling this more than once is safe
    async markMessagesAsRead() {
      if (!this.conversation || !this.conversation.messages) return;

      try {
        const requests = this.conversation.messages
          .filter((message) => !this.isMyMessage(message))
          .map((message) => api.put(`/messages/${message.id}/read`));

        await Promise.all(requests);
      } catch (e) {
        // Do not block the page if read tracking fails This is a secondary action, so just log it for debugging.
        console.error("Cannot mark messages as read:", e);
      }
    },


    // Deletes one of the current user's messages, then reloads the conversation.
    async deleteMessage(messageId) {
      if (!messageId) return;

      const confirmed = window.confirm("Do you want to delete this message?");
      if (!confirmed) return;

      this.errorMessage = "";

      try {
        await api.delete(`/messages/${messageId}`);

        // Reload the conversation so the UI reflects the deleted state.
        await this.loadConversation();
      } catch (e) {
        if (e.response && e.response.data && e.response.data.message) {
          this.errorMessage = e.response.data.message;
        } else {
          this.errorMessage = "Cannot delete message.";
        }
      }
    },

    // Adds a simple reaction/comment to a message, then reloads the conversation.
    async addComment(messageId, content) {
      if (!messageId || !content) return;

      this.errorMessage = "";

      try {
        await api.post(`/messages/${messageId}/comments`, {
          content,
        });

        // Reload the conversation so the new comment appears immediately.
        await this.loadConversation();
      } catch (e) {
        if (e.response && e.response.data && e.response.data.message) {
          this.errorMessage = e.response.data.message;
        } else {
          this.errorMessage = "Cannot add reaction.";
        }
      }
    },

    // Deletes one of the current user's comments from a message, then reloads the conversation.
    async deleteComment(messageId, commentId) {
      if (!messageId || !commentId) return;

      this.errorMessage = "";

      try {
        await api.delete(`/messages/${messageId}/comments/${commentId}`);

        // Reload the conversation so the removed comment disappears from the UI.
        await this.loadConversation();
      } catch (e) {
        if (e.response && e.response.data && e.response.data.message) {
          this.errorMessage = e.response.data.message;
        } else {
          this.errorMessage = "Cannot remove reaction.";
        }
      }
    },

    // Returns true if the currently opened conversation is a group.
    isGroupConversation() {
      return this.conversation && this.conversation.isGroup;
    },

    // Updates the current group's name, then reloads the conversation.
    async updateGroupName() {
      if (!this.conversation || !this.conversation.id) return;

      const newName = this.groupNameInput.trim();
      if (!newName) {
        this.errorMessage = "Group name must be a non-empty string.";
        return;
      }

      this.errorMessage = "";

      try {
        await api.put(`/groups/${this.conversation.id}/name`, {
          name: newName,
        });

        // Reload the conversation so the UI shows the updated name.
        await this.loadConversation();
      } catch (e) {
        if (e.response && e.response.data && e.response.data.message) {
          this.errorMessage = e.response.data.message;
        } else {
          this.errorMessage = "Cannot update group name.";
        }
      }
    },

    // Updates the current group's photo, then reloads the conversation.
    async updateGroupPhoto() {
      if (!this.conversation || !this.conversation.id) return;

      const newPhoto = this.groupPhotoInput.trim();
      if (!newPhoto) {
        this.errorMessage = "Group photo must be a non-empty string.";
        return;
      }

      this.errorMessage = "";

      try {
        await api.put(`/groups/${this.conversation.id}/photo`, {
          photo: newPhoto,
        });

        // Reload the conversation so the UI shows the updated photo.
        await this.loadConversation();
      } catch (e) {
        if (e.response && e.response.data && e.response.data.message) {
          this.errorMessage = e.response.data.message;
        } else {
          this.errorMessage = "Cannot update group photo.";
        }
      }
    },

    // Loads all users that are not already members of the current group.
    async loadAvailableUsersForGroup() {
      if (!this.isGroupConversation()) {
        this.availableUsers = [];
        return;
      }

      try {
        const response = await api.get("/users");
        const allUsers = response.data.items || [];

        // Build a set of user IDs that are already in the group.
        const memberIds = new Set(
          (this.conversation.members || []).map((member) => member.id)
        );

        // Keep only users that are not already part of the group.
        this.availableUsers = allUsers.filter((user) => !memberIds.has(user.id));
      } catch (e) {
        console.error("Cannot load available users for group:", e);
        this.availableUsers = [];
      }
    },

    // Adds the selected user to the current group, then reloads the conversation.
    async addUserToGroup() {
      if (!this.conversation || !this.conversation.id) return;

      const userId = this.selectedUserToAdd.trim();
      if (!userId) {
        this.errorMessage = "Select a user to add.";
        return;
      }

      this.errorMessage = "";

      try {
        await api.post(`/groups/${this.conversation.id}/members`, {
          userId,
        });

        // Clear the current selection.
        this.selectedUserToAdd = "";

        // Reload the conversation so the new member appears immediately.
        await this.loadConversation();
      } catch (e) {
        if (e.response && e.response.data && e.response.data.message) {
          this.errorMessage = e.response.data.message;
        } else {
          this.errorMessage = "Cannot add user to group.";
        }
      }
    },

    // Removes the current user from the group and redirects back to the conversations list.
    async leaveGroup() {
      if (!this.conversation || !this.conversation.id) return;

      const confirmed = window.confirm("Do you want to leave this group?");
      if (!confirmed) return;

      this.errorMessage = "";

      try {
        await api.delete(`/groups/${this.conversation.id}/members/me`);

        // After leaving the group, this user should no longer stay on its page.
        this.$router.push("/conversations");
      } catch (e) {
        if (e.response && e.response.data && e.response.data.message) {
          this.errorMessage = e.response.data.message;
        } else {
          this.errorMessage = "Cannot leave group.";
        }
      }
    },

    // Formats an ISO timestamp to be readable
    formatDateTime(value) {
      if (!value) return "";

      const date = new Date(value);
      if (Number.isNaN(date.getTime())) return value;

      return date.toLocaleString();
    },

  },
};
</script>