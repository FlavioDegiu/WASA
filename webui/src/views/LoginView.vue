<template>
  <div class="container py-5">
    <div class="row justify-content-center">
      <div class="col-12 col-md-6 col-lg-4">
        <div class="card shadow-sm">
          <div class="card-body">
            <h1 class="h3 mb-4 text-center">WASAText Login</h1>

            <ErrorMsg v-if="errorMessage" :msg="errorMessage" />

            <form @submit.prevent="doLogin">
              <div class="mb-3">
                <label for="username" class="form-label">Username</label>
                <input
                  id="username"
                  v-model.trim="username"
                  type="text"
                  class="form-control"
                  minlength="3"
                  maxlength="16"
                  required
                />
              </div>

              <button class="btn btn-primary w-100" type="submit" :disabled="loading">
                <span v-if="!loading">Login</span>
                <span v-else>Loading...</span>
              </button>
            </form>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import api from "../services/axios";
import ErrorMsg from "../components/ErrorMsg.vue";

export default {
  name: "LoginView",
  components: {
    ErrorMsg,
  },
  data() {
    return {
      username: "",
      loading: false,
      errorMessage: "",
    };
  },
  methods: {
    async doLogin() {
      this.errorMessage = "";

      if (this.username.length < 3 || this.username.length > 16) {
        this.errorMessage = "Username must be between 3 and 16 characters.";
        return;
      }

      this.loading = true;

      try {
        // Call the simplified login endpoint.
        const response = await api.post("/session", {
          name: this.username,
        });

        // Save the returned identifier to reuse it as Bearer token.
        localStorage.setItem("token", response.data.identifier);

        // Optional: keep the username locally too for UI convenience.
        localStorage.setItem("username", this.username);

        // After login, go to the conversations page.
        this.$router.push("/conversations");
      } catch (e) {
        if (e.response && e.response.data && e.response.data.message) {
          this.errorMessage = e.response.data.message;
        } else {
          this.errorMessage = "Login failed.";
        }
      } finally {
        this.loading = false;
      }
    },
  },
};
</script>