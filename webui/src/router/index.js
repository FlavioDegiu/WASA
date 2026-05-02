import {createRouter, createWebHashHistory} from 'vue-router'
import HomeView from '../views/HomeView.vue'
import LoginView from "../views/LoginView.vue";
import ConversationsView from "../views/ConversationsView.vue";
import ConversationView from "../views/ConversationView.vue";
import ProfileView from "../views/ProfileView.vue";
import NewConversationView from "../views/NewConversationView.vue";

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: "/",
      name: "login",
      component: LoginView,
    },
    {
      path: "/home",
      name: "home",
      component: HomeView,
    },
    {
      path: "/conversations",
      name: "conversations",
      component: ConversationsView,
    },
    {
      path: "/conversations/:conversationId",
      name: "conversation",
      component: ConversationView,
      props: true,
    },
    {
      path: "/profile",
      name: "profile",
      component: ProfileView,
    },
    {
    path: "/new-conversation",
    name: "new-conversation",
    component: NewConversationView,
    },
  ],
});

export default router
