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
          <div class="mb-3 d-flex justify-content-center">
            <img
              v-if="hasPhoto(user.photo)"
              :src="user.photo"
              alt="Profile"
              class="rounded-circle border"
              style="width: 96px; height: 96px; object-fit: cover;"
            />
            <div
              v-else
              class="rounded-circle border bg-light d-flex align-items-center justify-content-center text-muted"
              style="width: 96px; height: 96px;"
            >
              👤
            </div>
          </div>

          <p><strong>ID:</strong> {{ user.id }}</p>
          <p><strong>Username:</strong> {{ user.username }}</p>
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
        <label class="form-label">New profile photo</label>

        <input
          ref="profilePhotoInput"
          class="form-control mb-2"
          type="file"
          accept="image/*"
          @change="handleProfilePhotoSelected"
        />

        <div v-if="selectedProfilePhotoPreview" class="mb-3">
          <div class="small text-muted mb-1">Selected profile photo preview</div>
          <img
            :src="selectedProfilePhotoPreview"
            alt="Selected profile preview"
            class="img-fluid rounded border"
            style="max-height: 200px;"
          />

          <div class="mt-2">
            <button class="btn btn-sm btn-outline-secondary" type="button" @click="clearSelectedProfilePhoto">
              Remove selected image
            </button>
          </div>
        </div>

        <button class="btn btn-primary" type="submit">
          Update photo
        </button>
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
      selectedProfilePhotoFile: null,
  selectedProfilePhotoPreview: "",
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
    
    // Updates the current user's profile photo using the selected file.
    async updatePhoto() {
      this.errorMessage = "";

      try {
        let photoToSend = this.newPhoto.trim();

        // If a file was selected, convert it to a data URL and use that.
        if (this.selectedProfilePhotoFile) {
          photoToSend = await this.readSelectedProfilePhotoAsDataUrl();
        }

        if (!photoToSend) {
          this.errorMessage = "Select an image or provide a photo string.";
          return;
        }

        await api.put("/users/me/photo", { photo: photoToSend });

        this.newPhoto = "";
        this.clearSelectedProfilePhoto();
        await this.loadProfile();
      } catch (e) {
        if (e.response && e.response.data && e.response.data.message) {
          this.errorMessage = e.response.data.message;
        } else {
          this.errorMessage = "Cannot update photo.";
        }
      }
    },

    // Returns true if the given photo string looks usable for rendering.
    hasPhoto(photo) {
      return !!photo && photo.trim() !== "";
    },

    // Handles profile photo file selection and stores a preview.
    handleProfilePhotoSelected(event) {
      const file = event.target.files && event.target.files[0];
      if (!file) {
        this.selectedProfilePhotoFile = null;
        this.selectedProfilePhotoPreview = "";
        return;
      }

      if (!file.type.startsWith("image/")) {
        this.errorMessage = "Please select a valid image file.";
        this.selectedProfilePhotoFile = null;
        this.selectedProfilePhotoPreview = "";
        return;
      }

      this.errorMessage = "";
      this.selectedProfilePhotoFile = file;

      const reader = new FileReader();
      reader.onload = () => {
        this.selectedProfilePhotoPreview = reader.result;
      };
      reader.readAsDataURL(file);
    },

    // Clears the currently selected profile photo.
    clearSelectedProfilePhoto() {
      this.selectedProfilePhotoFile = null;
      this.selectedProfilePhotoPreview = "";

      if (this.$refs.profilePhotoInput) {
        this.$refs.profilePhotoInput.value = "";
      }
    },

    // Converts the selected profile photo to a data URL string.
    readSelectedProfilePhotoAsDataUrl() {
      return new Promise((resolve, reject) => {
        if (!this.selectedProfilePhotoFile) {
          resolve("");
          return;
        }

        const reader = new FileReader();
        reader.onload = () => resolve(reader.result);
        reader.onerror = () => reject(new Error("Cannot read selected profile photo"));
        reader.readAsDataURL(this.selectedProfilePhotoFile);
      });
    },
  },
};
</script>