
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
      <div class="mb-3 d-flex align-items-center gap-3">
        <!-- Conversation photo -->
        <div>
          <img
            v-if="hasPhoto(getConversationPhoto())"
            :src="getConversationPhoto()"
            alt="Conversation"
            class="rounded-circle border"
            style="width: 56px; height: 56px; object-fit: cover;"
          />
          <div
            v-else
            class="rounded-circle border bg-light d-flex align-items-center justify-content-center text-muted"
            style="width: 56px; height: 56px;"
          >
            👤
          </div>
        </div>

        <div>
          <h1 class="h3 mb-1">
            {{ getConversationTitle() }}
          </h1>
          <div class="text-muted">
            {{ getConversationSubtitle() }}
          </div>
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

            <input
              ref="groupPhotoInput"
              type="file"
              class="form-control mb-2"
              accept="image/*"
              @change="handleGroupPhotoSelected"
            />

            <div v-if="selectedGroupPhotoPreview" class="mb-3">
              <div class="small text-muted mb-1">Selected group photo preview</div>
              <img
                :src="selectedGroupPhotoPreview"
                alt="Selected group preview"
                class="img-fluid rounded border"
                style="max-height: 200px;"
              />

              <div class="mt-2">
                <button class="btn btn-sm btn-outline-secondary" type="button" @click="clearSelectedGroupPhoto">
                  Remove selected image
                </button>
              </div>
            </div>

            <button class="btn btn-outline-primary" type="submit">
              Update Photo
            </button>
          </form>

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
              class="list-group-item d-flex align-items-center gap-3"
            >
              <img
                v-if="hasPhoto(member.photo)"
                :src="member.photo"
                alt="Member"
                class="rounded-circle border"
                style="width: 40px; height: 40px; object-fit: cover;"
              />
              <div
                v-else
                class="rounded-circle border bg-light d-flex align-items-center justify-content-center text-muted"
                style="width: 40px; height: 40px;"
              >
                👤
              </div>

              <div>
                {{ member.username }}
                <span v-if="member.id === currentUserId" class="text-muted small">
                  (you)
                </span>
              </div>
            </li>
          </ul>
      </div>

      <div class="mb-4">
        <h2 class="h5">Messages</h2>
        <div v-if="conversation.messages.length === 0" class="text-muted">
          No messages yet.
        </div>

        <!-- messages -->
        <div
          v-for="message in conversation.messages"
          :key="message.id"
          class="card mb-2"
          :class="isMyMessage(message) ? 'border-primary' : 'border-light'"
        >
          <div class="card-body">
            
            <!-- message data (sender username and timestamp)-->
            <div class="d-flex justify-content-between align-items-center mb-1">
              <div class="small text-muted">
                {{ isMyMessage(message) ? "You" : message.senderUsername }}
                •
                {{ formatDateTime(message.createdAt) }}

                <span v-if="getMessageCheckmarks(message)" class="ms-2 text-primary fw-semibold">
                  {{ getMessageCheckmarks(message) }}
                </span>
              </div>
              
              <!-- buttons on the right off the message -->
              <div class="d-flex gap-2">
                <button
                  class="btn btn-sm btn-outline-secondary"
                  @click="startReply(message)"
                  :disabled="message.deleted"
                >
                  Reply
                </button>
                
                <button
                  class="btn btn-sm btn-outline-secondary"
                  @click="startForwarding(message.id)"
                  :disabled="message.deleted"
                >
                  Forward
                </button>

                <button
                  v-if="isMyMessage(message) && !message.deleted"
                  class="btn btn-sm btn-outline-danger"
                  @click="deleteMessage(message.id)"
                >
                  Delete
                </button>
              </div>

            </div>

            <!-- Forwarded marker -->
            <div v-if="isForwardedMessage(message)" class="small text-muted mb-1">
              <span class="badge text-bg-light border">Forwarded</span>
            </div>

            <!-- this message is replying to ... text box -->
            <div
              v-if="getRepliedMessage(message)"
              class="border-start border-3 ps-2 mb-2 small text-muted"
            >
              <div class="fw-semibold">
                Replying to {{ getRepliedMessage(message).senderUsername }}
              </div>
              <div>
                {{ getReplyPreviewContent(getRepliedMessage(message)) }}
              </div>
            </div>

            <!-- Message image -->
            <div v-if="message.type === 'image' && getImageMessageData(message)" class="mt-2">
              <img
                :src="getImageMessageData(message)"
                alt="Sent image"
                class="img-fluid rounded border"
                style="max-height: 300px;"
              />
            </div>

            <!-- message content on the left-->
            <div :class="message.deleted ? 'text-muted fst-italic' : ''">
              {{ getVisibleMessageContent(message) }}
            </div>

            <!-- comments buttons below the mesasge -->
            <div class="mt-2">
              <button
                class="btn btn-sm btn-outline-secondary"
                @click="addComment(message.id, '😀')"
                :disabled="message.deleted || hasMyReaction(message)"
              >
                😀
              </button>
            </div>

            <!-- display comments below comment buttons -->
            <div v-if="message.comments && message.comments.length > 0" class="mt-2">
              <div
                v-for="comment in message.comments"
                :key="comment.id"
                class="d-inline-flex align-items-center border rounded-pill px-2 py-1 me-2 mb-2 small bg-light"
              >
                <span class="me-2">
                  {{ comment.content }} {{ comment.authorUsername }}
                </span>

                <!-- button to delete my comment next to the comment -->
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

            <!-- Message forwarding section -->
            <div v-if="isForwardingThisMessage(message)" class="mt-3 border rounded p-3 bg-light">
              <div class="mb-2 fw-semibold">Forward message</div>

              <div class="mb-3">
                <label class="form-label">Choose an existing conversation</label>

                <div v-if="availableForwardConversations.length === 0" class="text-muted small">
                  No existing destination conversations available.
                </div>

                <select
                  v-else
                  v-model="selectedForwardConversationId"
                  class="form-select"
                >
                  <option value="">Select conversation</option>
                  <option
                    v-for="conv in availableForwardConversations"
                    :key="conv.id"
                    :value="conv.id"
                  >
                    {{ conv.title }}
                  </option>
                </select>
              </div>

              <div class="mb-3">
                <label class="form-label">Or forward to a user</label>

                <div v-if="availableForwardUsers.length === 0" class="text-muted small">
                  No users available.
                </div>

                <select
                  v-else
                  v-model="selectedForwardUserId"
                  class="form-select"
                >
                  <option value="">Select user</option>
                  <option
                    v-for="user in availableForwardUsers"
                    :key="user.id"
                    :value="user.id"
                  >
                    {{ user.username }}
                  </option>
                </select>
              </div>

              <div class="d-flex gap-2">
                <button class="btn btn-outline-primary" type="button" @click="confirmForwardMessage">
                  Confirm
                </button>

                <button class="btn btn-outline-secondary" type="button" @click="cancelForwarding">
                  Cancel
                </button>
              </div>
            </div>

          </div>
        </div>
      </div>

      <!-- Replying to message preview when writing a message -->
      <div v-if="replyingToMessage" class="border rounded p-2 mb-2 bg-light">
        <div class="d-flex justify-content-between align-items-start">
          <div>
            <div class="small fw-semibold">
              Replying to {{ replyingToMessage.senderUsername }}
            </div>
            <div class="small text-muted">
              {{ getReplyPreviewContent(replyingToMessage) }}
            </div>
          </div>

          <!-- Cancel reply button -->
          <button class="btn btn-sm btn-outline-secondary" type="button" @click="cancelReply">
            Cancel
          </button>
        </div>
      </div>
      
      <!-- Attach image button -->
      <div class="mb-2">
        <label class="form-label">Attach image</label>
        <input
          ref="imageInput"
          type="file"
          class="form-control"
          accept="image/*"
          @change="handleImageSelected"
        />
      </div>

      <!-- Send form -->
      <form @submit.prevent="sendMessage">
        <div class="input-group">          
          <input
            v-model.trim="newMessage"
            type="text"
            class="form-control"
            placeholder="Write a message..."
            maxlength="4096"
          />



          <!-- Send button -->
          <button
            class="btn btn-primary"
            type="submit"
            :disabled="sending || (!newMessage.trim() && !selectedImageFile)"
          >
            Send
          </button>
        </div>
      </form>

      <!-- Picked image show -->
      <div v-if="selectedImagePreview" class="mb-3">
        <div class="small text-muted mb-1">Selected image preview</div>
        <img
          :src="selectedImagePreview"
          alt="Selected preview"
          class="img-fluid rounded border"
          style="max-height: 200px;"
        />

        <div class="mt-2">
          <button class="btn btn-sm btn-outline-secondary" type="button" @click="clearSelectedImage">
            Remove image
          </button>
        </div>
      </div>

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
      refreshIntervalId: null,
      forwardingMessageId: "",
      availableForwardConversations: [],
      selectedForwardConversationId: "",
      availableForwardUsers: [],
      selectedForwardUserId: "",
      replyingToMessage: null,
      selectedImageFile: null,
      selectedImagePreview: "",
      selectedGroupPhotoFile: null,
      selectedGroupPhotoPreview: "",
    };
  },

  async mounted() {
    await this.loadConversation();
    this.startAutoRefresh();
  },
  beforeUnmount() {
    this.stopAutoRefresh();
  },

  watch: {
    selectedForwardConversationId(newValue) {
      if (newValue) {
        this.selectedForwardUserId = "";
      }
    },
    selectedForwardUserId(newValue) {
      if (newValue) {
        this.selectedForwardConversationId = "";
      }
    },
  },

  methods: {
    // Returns the text that should be shown for a message.
    // Deleted messages keep their original content in the database, but the UI hides it from the user
    getVisibleMessageContent(message) {
      if (message.deleted) {
        return "[deleted message]";
      }

      if (message.type === "image") {
        const parsed = this.parseImageMessageContent(message);
        return parsed && parsed.text ? parsed.text : "";
      }

      return message.content;
    },
    

    // Loads the current conversation and refreshes read status.
    async loadConversation(background = false) {
      if (!background) {
        this.loading = true;
      }

      this.errorMessage = "";

      try {
        // Load the current conversation and its messages.
        const response = await api.get(`/conversations/${this.conversationId}`);
        this.conversation = response.data;

        // Mark incoming messages as read.
        await this.markMessagesAsRead();

        // Reload once so the UI shows updated read status and comments.
        const refreshedResponse = await api.get(`/conversations/${this.conversationId}`);
        this.conversation = refreshedResponse.data;

        // Keep editable group fields in sync with the loaded conversation.
        this.groupNameInput = this.conversation.name || "";

        // Refresh the list of users that can still be added to this group.
        await this.loadAvailableUsersForGroup();
      } catch (e) {
        if (e.response && e.response.data && e.response.data.message) {
          this.errorMessage = e.response.data.message;
        } else {
          this.errorMessage = "Cannot load conversation.";
        }
      } finally {
        if (!background) {
          this.loading = false;
        }
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
    
    
    // Sends either a normal text message or an image message with optional caption.
    async sendMessage() {
      const trimmedText = this.newMessage.trim();

      // Do not send empty messages if there is no selected image.
      if (!trimmedText && !this.selectedImageFile) return;

      this.sending = true;
      this.errorMessage = "";
      this.stopAutoRefresh();

      try {
        let payload;

        if (this.selectedImageFile) {
          // Read the image as base64 data URL and pack it together with the caption.
          const imageData = await this.readSelectedImageAsDataUrl();

          payload = {
            type: "image",
            content: JSON.stringify({
              text: trimmedText,
              imageData: imageData,
            }),
            replyToMessageId: this.replyingToMessage ? this.replyingToMessage.id : "",
          };
        } else {
          payload = {
            type: "text",
            content: trimmedText,
            replyToMessageId: this.replyingToMessage ? this.replyingToMessage.id : "",
          };
        }

        await api.post(`/conversations/${this.conversationId}/messages`, payload);

        // Reset composer state after success.
        this.newMessage = "";
        this.replyingToMessage = null;
        this.clearSelectedImage();

        // Also clear the native file input element if present.
        if (this.$refs.imageInput) {
          this.$refs.imageInput.value = "";
        }

        await this.loadConversation();
      } catch (e) {
        if (e.response && e.response.data && e.response.data.message) {
          this.errorMessage = e.response.data.message;
        } else {
          this.errorMessage = "Cannot send message.";
        }
      } finally {
        this.sending = false;
        this.startAutoRefresh();
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

      this.errorMessage = "";

      try {
        let photoToSend = this.groupPhotoInput.trim();

        // If a file was selected, prefer it over the plain text field.
        if (this.selectedGroupPhotoFile) {
          photoToSend = await this.readSelectedGroupPhotoAsDataUrl();
        }

        if (!photoToSend) {
          this.errorMessage = "Select an image or provide a group photo string.";
          return;
        }

        await api.put(`/groups/${this.conversation.id}/photo`, {
          photo: photoToSend,
        });

        this.clearSelectedGroupPhoto();
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

    // Starts automatic refresh of the opened conversation.
    startAutoRefresh() {
      // Avoid duplicate timers.
      this.stopAutoRefresh();

      // Refresh every 3 seconds.
      this.refreshIntervalId = setInterval(() => {
        this.loadConversation(true);
      }, 3000);
    },

    // Stops the automatic refresh timer if it exists.
    stopAutoRefresh() {
      if (this.refreshIntervalId) {
        clearInterval(this.refreshIntervalId);
        this.refreshIntervalId = null;
      }
    },

    // Returns the checkmark string to show for one of the current user's messages.
    getMessageCheckmarks(message) {
      // Received messages must not show any checkmarks.
      if (!this.isMyMessage(message)) {
        return "";
      }

      if (message.status && message.status.readByAll) {
        return "✓✓";
      }

      if (message.status && message.status.deliveredToAll) {
        return "✓";
      }

      return "";
    },

    // Returns true if the current user already reacted to this message
    hasMyReaction(message) {
      if (!message.comments) return false;

      return message.comments.some(
        (comment) => comment.authorId === this.currentUserId
      );
    },
    
    // Returns true if the message was created by forwarding another message
    isForwardedMessage(message) {
      return !!message.forwardedFromMessageId;
    },

    // Starts the forwarding flow for the selected message.
    async startForwarding(messageId) {
      this.errorMessage = "";
      this.forwardingMessageId = messageId;
      this.selectedForwardConversationId = "";
      this.selectedForwardUserId = "";

      await Promise.all([
        this.loadAvailableForwardConversations(),
        this.loadAvailableForwardUsers(),
      ]);
    },

    // Cancels the current forwarding action.
    cancelForwarding() {
      this.forwardingMessageId = "";
      this.selectedForwardConversationId = "";
      this.selectedForwardUserId = "";
      this.availableForwardConversations = [];
      this.availableForwardUsers = [];
    },

    // Loads all conversations that can be used as forwarding destinations, excluding the currently open one
    async loadAvailableForwardConversations() {
      try {
        const response = await api.get("/conversations");
        const items = response.data.items || [];

        // Exclude the current conversation to keep the UI simpler.
        this.availableForwardConversations = items.filter(
          (conv) => conv.id !== this.conversationId
        );
      } catch (e) {
        console.error("Cannot load forwarding destinations:", e);
        this.availableForwardConversations = [];
      }
    },

    // Loads all users except the currently logged-in one. These users can be used as forwarding targets even if no direct conversation exists yet.
    async loadAvailableForwardUsers() {
      try {
        const response = await api.get("/users");
        const items = response.data.items || [];

        this.availableForwardUsers = items.filter(
          (user) => user.id !== this.currentUserId
        );
      } catch (e) {
        console.error("Cannot load forwarding users:", e);
        this.availableForwardUsers = [];
      }
    },

    // Forwards the selected message to an existing conversation or to a user (creating a direct conversation first if needed).
    async confirmForwardMessage() {
      if (!this.forwardingMessageId) return;

      this.errorMessage = "";

      try {
        let destinationConversationId = this.selectedForwardConversationId.trim();

        // If no existing conversation was selected, try forwarding to a user.
        if (!destinationConversationId) {
          const selectedUserId = this.selectedForwardUserId.trim();

          if (!selectedUserId) {
            this.errorMessage = "Select a destination conversation or user.";
            return;
          }

          // Create a direct conversation with that user first.
          destinationConversationId = await this.createDirectConversationForForward(selectedUserId);
        }

        await api.post(`/messages/${this.forwardingMessageId}/forward`, {
          conversationId: destinationConversationId,
        });

        // Keep the destination so we can optionally open it after forwarding.
        const finalDestinationId = destinationConversationId;

        // Clear forwarding state.
        this.cancelForwarding();

        // Open the destination conversation so the forwarded message is immediately visible.
        this.$router.push(`/conversations/${finalDestinationId}`);
      } catch (e) {
        if (e.response && e.response.data && e.response.data.message) {
          this.errorMessage = e.response.data.message;
        } else {
          this.errorMessage = "Cannot forward message.";
        }
      }
    },

    // Returns true if this message is currently selected for forwarding
    isForwardingThisMessage(message) {
      return this.forwardingMessageId === message.id;
    },

    // Creates a direct conversation with the given user and returns its ID.
    async createDirectConversationForForward(userId) {
      const response = await api.post("/conversations", {
        isGroup: false,
        members: [userId],
      });

      return response.data.id;
    },

    // Starts replying to the selected message.
    startReply(message) {
      this.replyingToMessage = message;
    },

    // Cancels the current reply.
    cancelReply() {
      this.replyingToMessage = null;
    },

    // Returns the original referenced message for a reply, if it exists in the loaded conversation.
    getRepliedMessage(message) {
      if (!message.replyToMessageId || !this.conversation || !this.conversation.messages) {
        return null;
      }

      return this.conversation.messages.find(
        (candidate) => candidate.id === message.replyToMessageId
      ) || null;
    },

    // Returns the short preview text used in reply snippets.
    // Can't make this work. I'm going insane
    getReplyPreviewContent(message) {
      if (!message) return "";

      if (message.type === "image") return "[image]";
      if (message.type === "gif") return "[gif]";
      if (message.deleted) return "[deleted message]";
      if (!message.content) return "";

      const maxLength = 30;
      
      if (message.content.length <= maxLength - 8 && message.type === "image") { return `[image] ` + message.content}
      if (message.content.length <= maxLength - 6 && message.type === "gif") { return `[gif] ` + message.content}
      if (message.content.length <= maxLength && message.type !== "gif" && message.type != "image") { return message.content; }

      if (message.type === "image") return `[image] ` + message.content.slice(0, maxLength - 11) + "...";
      if (message.type === "gif") return `[gif] ` + message.content.slice(0, maxLength - 9) + "...";
      return message.content.slice(0, maxLength - 3) + "...";
    },

    // Handles image file selection and stores a local preview.
    handleImageSelected(event) {
      const file = event.target.files && event.target.files[0];
      if (!file) {
        this.selectedImageFile = null;
        this.selectedImagePreview = "";
        return;
      }

      // Only allow image files.
      if (!file.type.startsWith("image/")) {
        this.errorMessage = "Please select a valid image file.";
        this.selectedImageFile = null;
        this.selectedImagePreview = "";
        return;
      }

      this.errorMessage = "";
      this.selectedImageFile = file;

      // Build a local preview so the user can see what will be sent.
      const reader = new FileReader();
      reader.onload = () => {
        this.selectedImagePreview = reader.result;
      };
      reader.readAsDataURL(file);
    },

    // Clears the currently selected image and resets the file input preview state.
    clearSelectedImage() {
      this.selectedImageFile = null;
      this.selectedImagePreview = "";
    },

    // Converts the selected image file into a data URL string.
    readSelectedImageAsDataUrl() {
      return new Promise((resolve, reject) => {
        if (!this.selectedImageFile) {
          resolve("");
          return;
        }

        const reader = new FileReader();

        reader.onload = () => resolve(reader.result);
        reader.onerror = () => reject(new Error("Cannot read selected image file"));

        reader.readAsDataURL(this.selectedImageFile);
      });
    },

    // Safely parses the structured content of an image message.
    parseImageMessageContent(message) {
      if (!message || message.type !== "image" || !message.content) {
        return null;
      }

      try {
        return JSON.parse(message.content);
      } catch (e) {
        return null;
      }
    },

    // Returns the image data URL for an image message, if available.
    getImageMessageData(message) {
      if (message.deleted || message.type !== "image") {
        return "";
      }

      const parsed = this.parseImageMessageContent(message);
      return parsed && parsed.imageData ? parsed.imageData : "";
    },

    // Returns true if the current conversation has a renderable photo string.
    hasPhoto(photo) {
      return !!photo && photo.trim() !== "";
    },

    // Returns the photo that should be shown in the conversation header.
    // For groups, use the group photo.
    // For direct conversations, use the other user's photo.
    getConversationPhoto() {
      if (!this.conversation) return "";

      if (this.conversation.isGroup) {
        return this.conversation.photo || "";
      }

      const otherMember = (this.conversation.members || []).find(
        (member) => member.id !== this.currentUserId
      );

      return otherMember && otherMember.photo ? otherMember.photo : "";
    },

    // Handles group photo file selection and stores a preview.
    handleGroupPhotoSelected(event) {
      const file = event.target.files && event.target.files[0];
      if (!file) {
        this.selectedGroupPhotoFile = null;
        this.selectedGroupPhotoPreview = "";
        return;
      }

      if (!file.type.startsWith("image/")) {
        this.errorMessage = "Please select a valid image file.";
        this.selectedGroupPhotoFile = null;
        this.selectedGroupPhotoPreview = "";
        return;
      }

      this.errorMessage = "";
      this.selectedGroupPhotoFile = file;

      const reader = new FileReader();
      reader.onload = () => {
        this.selectedGroupPhotoPreview = reader.result;
      };
      reader.readAsDataURL(file);
    },

    // Clears the selected group photo.
    clearSelectedGroupPhoto() {
      this.selectedGroupPhotoFile = null;
      this.selectedGroupPhotoPreview = "";

      if (this.$refs.groupPhotoInput) {
        this.$refs.groupPhotoInput.value = "";
      }
    },

    // Converts the selected group photo into a data URL string.
    readSelectedGroupPhotoAsDataUrl() {
      return new Promise((resolve, reject) => {
        if (!this.selectedGroupPhotoFile) {
          resolve("");
          return;
        }

        const reader = new FileReader();
        reader.onload = () => resolve(reader.result);
        reader.onerror = () => reject(new Error("Cannot read selected group photo"));
        reader.readAsDataURL(this.selectedGroupPhotoFile);
      });
    },

  },
};
</script>