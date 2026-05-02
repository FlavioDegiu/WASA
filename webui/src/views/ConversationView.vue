
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
      <h1 class="h3 mb-3">
        {{ conversation.name || "Conversation" }}
      </h1>

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
            <div class="small text-muted mb-1">
              {{ isMyMessage(message) ? "You" : message.senderUsername }} • {{ message.createdAt }}
            </div>
            <div>{{ message.content }}</div>
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
    };
  },
  async mounted() {
    await this.loadConversation();
  },
  methods: {
    
    
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
  },
};
</script>