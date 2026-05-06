<template>
  <div class="container py-4">
    <div class="d-flex justify-content-between align-items-center mb-4">
      <h1 class="h3 mb-0">Conversations</h1>
      <div class="d-flex gap-2">
        <router-link to="/profile" class="btn btn-outline-secondary btn-sm">
          Profile
        </router-link>
        <button class="btn btn-outline-danger btn-sm" @click="logout">
          Logout
        </button>
        <router-link to="/new-conversation" class="btn btn-primary btn-sm">
          New Conversation
        </router-link>
      </div>
    </div>

    <ErrorMsg v-if="errorMessage" :msg="errorMessage" />

    <LoadingSpinner v-if="loading" />

    <div v-else class="list-group">
      <router-link
        v-for="conversation in conversations"
        :key="conversation.id"
        :to="`/conversations/${conversation.id}`"
        class="list-group-item list-group-item-action"
      >
        <div class="d-flex w-100 justify-content-between align-items-start">
          <div>
            <h5 class="mb-1">{{ conversation.title }}</h5>
            <p class="mb-1 text-muted">
              {{ formatPreview(conversation.lastMessage) }}
            </p>
          </div>
          <small class="text-muted">
            {{ formatDateTime(conversation.updatedAt) }}
          </small>
        </div>
      </router-link>

      <div v-if="conversations.length === 0" class="text-muted">
        No conversations yet.
      </div>
    </div>
  </div>
</template>

<script>
import api from "../services/axios";
import ErrorMsg from "../components/ErrorMsg.vue";
import LoadingSpinner from "../components/LoadingSpinner.vue";

export default {
  name: "ConversationsView",
  components: {
    ErrorMsg,
    LoadingSpinner,
  },
  data() {
    return {
      conversations: [],
      loading: false,
      errorMessage: "",
    };
  },
  async mounted() {
    await this.loadConversations();
  },
  methods: {

    async loadConversations() {
      this.loading = true;
      this.errorMessage = "";

      try {
        // Load all conversations of the authenticated user.
        const response = await api.get("/conversations");
        this.conversations = response.data.items || [];
      } catch (e) {
        if (e.response && e.response.status === 401) {
          this.logout();
          return;
        }

        if (e.response && e.response.data && e.response.data.message) {
          this.errorMessage = e.response.data.message;
        } else {
          this.errorMessage = "Cannot load conversations.";
        }
      } finally {
        this.loading = false;
      }
    },

    formatPreview(lastMessage) {
      if (!lastMessage) return "";

      if (lastMessage.type === "image") return "[image]";
      if (lastMessage.type === "gif") return "[gif]";
      if (!lastMessage.content) return "";

      return lastMessage.content;
    },

    logout() {
      localStorage.removeItem("token");
      localStorage.removeItem("username");
      this.$router.push("/");
    },

    // Formats an ISO date string to be readable.
    formatDateTime(value) {
      if (!value) return "";

      const date = new Date(value);
      if (Number.isNaN(date.getTime())) return value;

      return date.toLocaleString();
    },


  },
};
</script>