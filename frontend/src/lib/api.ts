import axios from "axios";

const api = axios.create({
  baseURL: "/",
});

api.interceptors.request.use((config) => {
  const token = localStorage.getItem("token");
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

api.interceptors.response.use(
  (res) => res,
  (err) => {
    const url = err.config?.url ?? "";
    const is401 = err.response?.status === 401;

    // don't logout
    const skipLogout = ["/api/auth/password", "/api/login"].some((path) =>
      url.includes(path),
    );

    if (is401 && !skipLogout) {
      localStorage.removeItem("token");
      window.location.href = "/login";
    }

    return Promise.reject(err);
  },
);
export default api;
