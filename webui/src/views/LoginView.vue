<template>
  <div class="container py-5">
    <div class="row justify-content-center">
      <div class="col-12 col-md-6 col-lg-4">
        <div class="card shadow-sm">
          <div class="card-body">

            <!-- Page title -->
            <h1 class="h3 mb-4 text-center">WASAText Login</h1>

            <!-- Error message component (if not empty, will display an error) -->
            <ErrorMsg v-if="errorMessage" :msg="errorMessage" />

            <!--
            Submission form, binded to the doLogin JS func 
            (prevent stops the browser from standard behaviour)
            -->
            <form @submit.prevent="doLogin">
              <div class="mb-3">
                <label for="username" class="form-label">Username</label>
                <!-- bind username variable to field -->
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

              <!-- disable the Login button when the page state has loading = True -->
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
        // Call the simplified login endpoint using axios (api.post).
        const response = await api.post("/session", {
          name: this.username,
        });

        /*
        The application uses local storage
        so multiple pages of the same app will always have the same user logged in
        */

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