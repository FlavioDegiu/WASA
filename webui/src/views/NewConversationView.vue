<template>
  <div class="container py-4">
    <div class="mb-3">
      <router-link to="/conversations" class="btn btn-outline-secondary btn-sm">
        Back
      </router-link>
    </div>

    <h1 class="h3 mb-4">New Conversation</h1>

    <ErrorMsg v-if="errorMessage" :msg="errorMessage" />
    <LoadingSpinner v-if="loading" />

    <div v-else>
      <!-- Direct conversation section -->
      <div class="card mb-4">
        <div class="card-body">
          <h2 class="h5 mb-3">Start a direct conversation</h2>

          <div v-if="users.length === 0" class="text-muted">
            No users available.
          </div>

          <div class="list-group">
            <button
              v-for="user in users"
              :key="user.id"
              class="list-group-item list-group-item-action"
              @click="createDirectConversation(user.id)"
            >
              {{ user.username }}
            </button>
          </div>
        </div>
      </div>

      <!-- Group conversation section -->
      <div class="card">
        <div class="card-body">
          <h2 class="h5 mb-3">Create a group</h2>

          <div class="mb-3">
            <label class="form-label">Group name</label>
            <input
              v-model.trim="groupName"
              type="text"
              class="form-control"
              placeholder="Enter group name"
            />
          </div>

          <div class="mb-3">
            <label class="form-label">Select users</label>

            <div v-if="users.length === 0" class="text-muted">
              No users available.
            </div>

            <div
              v-for="user in users"
              :key="user.id"
              class="form-check"
            >
              <input
                :id="`user-${user.id}`"
                v-model="selectedUserIds"
                :value="user.id"
                class="form-check-input"
                type="checkbox"
              />
              <label class="form-check-label" :for="`user-${user.id}`">
                {{ user.username }}
              </label>
            </div>
          </div>

          <button class="btn btn-primary" @click="createGroupConversation">
            Create Group
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
  name: "NewConversationView",
  components: {
    ErrorMsg,
    LoadingSpinner,
  },
  data() {
    return {
      users: [],
      loading: false,
      errorMessage: "",
      groupName: "",
      selectedUserIds: [],
      currentUserId: localStorage.getItem("token") || "",
    };
  },
  async mounted() {
    await this.loadUsers();
  },
  methods: {
    // Loads all users except the currently authenticated one.
    async loadUsers() {
      this.loading = true;
      this.errorMessage = "";

      try {
        const response = await api.get("/users");
        const items = response.data.items || [];

        this.users = items.filter((user) => user.id !== this.currentUserId);
      } catch (e) {
        if (e.response && e.response.data && e.response.data.message) {
          this.errorMessage = e.response.data.message;
        } else {
          this.errorMessage = "Cannot load users.";
        }
      } finally {
        this.loading = false;
      }
    },

    // Creates a direct conversation with exactly one other user.
    async createDirectConversation(userId) {
      this.errorMessage = "";

      try {
        const response = await api.post("/conversations", {
          isGroup: false,
          members: [userId],
        });

        this.$router.push(`/conversations/${response.data.id}`);
      } catch (e) {
        if (e.response && e.response.data && e.response.data.message) {
          this.errorMessage = e.response.data.message;
        } else {
          this.errorMessage = "Cannot create direct conversation.";
        }
      }
    },

    // Creates a group conversation with a name and one or more selected users.
    async createGroupConversation() {
      this.errorMessage = "";

      if (!this.groupName) {
        this.errorMessage = "Group name is required.";
        return;
      }

      if (this.selectedUserIds.length === 0) {
        this.errorMessage = "Select at least one user.";
        return;
      }

      try {
        const response = await api.post("/conversations", {
          isGroup: true,
          name: this.groupName,
          members: this.selectedUserIds,
        });

        this.$router.push(`/conversations/${response.data.id}`);
      } catch (e) {
        if (e.response && e.response.data && e.response.data.message) {
          this.errorMessage = e.response.data.message;
        } else {
          this.errorMessage = "Cannot create group conversation.";
        }
      }
    },
  },
};
</script>