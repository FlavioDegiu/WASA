<template>
  <div class="container py-4">
    <div class="mb-3">
      <router-link to="/conversations" class="btn btn-outline-secondary btn-sm">
        Back
      </router-link>
    </div>

    <h1 class="h3 mb-4">My Profile</h1>

    <ErrorMsg v-if="errorMessage" :msg="errorMessage" />
    <LoadingSpinner v-if="loading" />

    <div v-else-if="user">
      <div class="card mb-4">
        <div class="card-body">
          <p><strong>ID:</strong> {{ user.id }}</p>
          <p><strong>Username:</strong> {{ user.username }}</p>
          <p><strong>Photo:</strong> {{ user.photo || "No photo" }}</p>
        </div>
      </div>

      <form class="mb-4" @submit.prevent="updateName">
        <label class="form-label">New username</label>
        <div class="input-group">
          <input v-model.trim="newName" class="form-control" type="text" />
          <button class="btn btn-primary" type="submit">Update name</button>
        </div>
      </form>

      <form @submit.prevent="updatePhoto">
        <label class="form-label">New photo path</label>
        <div class="input-group">
          <input v-model.trim="newPhoto" class="form-control" type="text" />
          <button class="btn btn-primary" type="submit">Update photo</button>
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
  name: "ProfileView",
  components: {
    ErrorMsg,
    LoadingSpinner,
  },
  data() {
    return {
      user: null,
      loading: false,
      errorMessage: "",
      newName: "",
      newPhoto: "",
    };
  },
  async mounted() {
    await this.loadProfile();
  },
  methods: {
    async loadProfile() {
      this.loading = true;
      this.errorMessage = "";

      try {
        const response = await api.get("/users/me");
        this.user = response.data;
      } catch (e) {
        if (e.response && e.response.data && e.response.data.message) {
          this.errorMessage = e.response.data.message;
        } else {
          this.errorMessage = "Cannot load profile.";
        }
      } finally {
        this.loading = false;
      }
    },
    async updateName() {
      if (!this.newName) return;

      try {
        await api.put("/users/me/name", { name: this.newName });
        this.newName = "";
        await this.loadProfile();
      } catch (e) {
        if (e.response && e.response.data && e.response.data.message) {
          this.errorMessage = e.response.data.message;
        } else {
          this.errorMessage = "Cannot update username.";
        }
      }
    },
    async updatePhoto() {
      if (!this.newPhoto) return;

      try {
        await api.put("/users/me/photo", { photo: this.newPhoto });
        this.newPhoto = "";
        await this.loadProfile();
      } catch (e) {
        if (e.response && e.response.data && e.response.data.message) {
          this.errorMessage = e.response.data.message;
        } else {
          this.errorMessage = "Cannot update photo.";
        }
      }
    },
  },
};
</script>