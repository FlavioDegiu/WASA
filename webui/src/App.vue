<script setup>
import { RouterLink, RouterView, useRouter } from 'vue-router'

// We use the router here so the logout button can redirect the user.
const router = useRouter()

// Clears the local session and sends the user back to login.
function logout() {
  localStorage.removeItem('token')
  localStorage.removeItem('username')
  router.push('/')
}
</script>

<template>
  <!-- Top navigation bar -->
  <header class="navbar navbar-dark sticky-top bg-dark flex-md-nowrap p-0 shadow">
    <a class="navbar-brand col-md-3 col-lg-2 me-0 px-3 fs-6" href="#/">WASAText</a>

    <!-- Button used by Bootstrap to open the sidebar on small screens -->
    <button
      class="navbar-toggler position-absolute d-md-none collapsed"
      type="button"
      data-bs-toggle="collapse"
      data-bs-target="#sidebarMenu"
      aria-controls="sidebarMenu"
      aria-expanded="false"
      aria-label="Toggle navigation"
    >
      <span class="navbar-toggler-icon"></span>
    </button>

    <!-- Right side actions -->
    <div class="navbar-nav">
      <div class="nav-item text-nowrap">
        <button class="btn btn-sm btn-outline-light me-3" @click="logout">
          Logout
        </button>
      </div>
    </div>
  </header>

  <div class="container-fluid">
    <div class="row">
      <!-- Sidebar -->
      <nav id="sidebarMenu" class="col-md-3 col-lg-2 d-md-block bg-light sidebar collapse">
        <div class="position-sticky pt-3 sidebar-sticky">
          <h6
            class="sidebar-heading d-flex justify-content-between align-items-center px-3 mt-4 mb-1 text-muted text-uppercase"
          >
            <span>WASAText</span>
          </h6>

          <ul class="nav flex-column">
            <li class="nav-item">
              <RouterLink to="/" class="nav-link">
                <svg class="feather">
                  <use href="/feather-sprite-v4.29.0.svg#log-in" />
                </svg>
                Login
              </RouterLink>
            </li>

            <li class="nav-item">
              <RouterLink to="/conversations" class="nav-link">
                <svg class="feather">
                  <use href="/feather-sprite-v4.29.0.svg#message-circle" />
                </svg>
                Conversations
              </RouterLink>
            </li>

            <li class="nav-item">
              <RouterLink to="/new-conversation" class="nav-link">
                <svg class="feather">
                  <use href="/feather-sprite-v4.29.0.svg#plus-circle" />
                </svg>
                New Conversation
              </RouterLink>
            </li>

            <li class="nav-item">
              <RouterLink to="/profile" class="nav-link">
                <svg class="feather">
                  <use href="/feather-sprite-v4.29.0.svg#user" />
                </svg>
                Profile
              </RouterLink>
            </li>
          </ul>
        </div>
      </nav>

      <!-- Main page content -->
      <main class="col-md-9 ms-sm-auto col-lg-10 px-md-4 py-4">
        <RouterView />
      </main>
    </div>
  </div>
</template>

<style>
/* Small improvement so the logout button so it blends better in the navbar */
.btn-outline-light {
  border-width: 1px;
}
</style>