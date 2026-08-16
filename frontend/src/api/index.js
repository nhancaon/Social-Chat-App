import axios from "axios";
import config from '@/runtime-config'

const API = axios.create({ baseURL: config.API_URL })
// attach the JWT to every outgoing request, if the user is signed in
API.interceptors.request.use((req) => {
  if (localStorage.getItem('profile')) {
    req.headers.Authorization = `Bearer ${JSON.parse(localStorage.getItem('profile')).token}`;
  }
  return req;
})

// user end points 
export const signIn = (formData) => API.post('user/signin', formData);
export const signUp = (formData) => API.post('user/signup', formData);
export const refreshUser = () => API.get('user/refresh');
export const fetchUserProfile = (id, page = 1) => API.get(`/user/getUser/${id}?page=${page}`);
export const getSuggestedUsers = (id) => API.get(`/user/getSug?id=${id}`);
export const updateUser = (userData) => API.patch(`user/Update/${userData._id}`, userData);
export const followUser = (id) => API.patch(`/user/${id}/following`);
export const fetchUsersByIds = (ids) => API.post('/user/batch', { ids });


// posts methods
export const fetchPost = (id) => API.get(`posts/${id}`);
export const fetchPosts = (page, id) => API.get(`posts?page=${page}&id=${id}`);
export const createPost = (newPost) => API.post('posts', newPost);
export const likePost = (id) => API.patch(`posts/${id}/likePost`);
export const commentPost = (value, id) => API.post(`posts/${id}/commentPost`, { value });
export const deleteComment = (postId, commentId) => API.delete(`posts/${postId}/comments/${commentId}`);
export const updatePost = (id, updatedPost) => API.patch(`posts/${id}`, updatedPost)
export const deletePost = (id) => API.delete(`posts/${id}`)
export const fetchPostsUsersBySearch = (searchQuery) => API.get(`posts/search?searchQuery=${searchQuery}`);


//  Notification
export const getNotificationForUser = (id) => API.get(`notification/${id}`)
export const markNotificationAsRead = (id) => API.get(`notification/mark-notification-asreaded?id=${id}`)

// chat -
export const sendMessage = (msg) => API.post('chat/sendmessage', msg);
export const getUnreadMessageCount = (id) => API.get(`chat/get-user-unreadmsg?userid=${id}`)
export const getMessagesByPage = (from, firstuid, seconduid) => API.get(`chat/getmsgsbynums?from=${from}&firstuid=${firstuid}&seconduid=${seconduid}`)
export const markMessageAsRead = (mainuid, otheruid) => API.get(`chat/mark-msg-asreaded?mainuid=${mainuid}&otheruid=${otheruid}`)