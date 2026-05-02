import axios from "axios";

const api = axios.create({
	baseURL: __API_URL__,
	timeout: 1000 * 5
});

// Add the Authorization header automatically if a token exists
api.interceptors.request.use((config) => {
  const token = localStorage.getItem("token");
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

export default api;
