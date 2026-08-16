import * as api from '../api/index.js';

function initialState() {
  return {
    isLoading: true,
    post: null,
    posts: [],
    searchResults: [],
  }
}

const Posts = {
  namespaced: true,
  state: initialState(),

  getters: {
    getCurrentPost: (state) => state.post,
    getAllPosts: (state) => state.posts,
    getSearchResults: (state) => state.searchResults,
    isLoading: (state) => state.isLoading,
  },

  mutations: {
    SET_LOADING(state, payload) {
      state.isLoading = payload;
    },
    SET_POST(state, payload) {
      state.post = payload;
    },
    SET_POSTS(state, payload) {
      state.posts = payload.data;
    },
    ADD_POST(state, payload) {
      state.posts.unshift(payload);
    },
    UPDATE_POST_IN_LIST(state, payload) {
      const index = state.posts.findIndex((p) => p._id === payload._id);
      if (index !== -1) {
        state.posts.splice(index, 1, payload);
      }
    },
    REMOVE_POST(state, id) {
      state.posts = state.posts.filter((p) => p._id !== id);
    },
    SET_SEARCH(state, payload) {
      state.searchResults = payload;
    },
    RESET_STATE(state) {
      Object.assign(state, initialState())
    },
  },

  actions: {
    async getPost({ commit }, id) {
      try {
        commit('SET_LOADING', true);
        const { data } = await api.fetchPost(id);
        commit('SET_POST', data);
        return data;
      } catch (error) {
        console.error('getPost error:', error);
        throw error;
      } finally {
        commit('SET_LOADING', false);
      }
    },

    async getPosts({ commit, rootGetters }, page) {
      const userId = rootGetters['auth/currentUserId'];
      if (!userId) {
        console.warn('getPosts: no userId found, skipping fetch');
        return;
      }
      try {
        commit('SET_LOADING', true);
        const { data } = await api.fetchPosts(page, userId);
        commit('SET_POSTS', data);
        return data;
      } catch (error) {
        console.error('getPosts error:', error);
        throw error;
      } finally {
        commit('SET_LOADING', false);
      }
    },

    // dùng chung cho tìm kiếm users & posts
    async getPostsUsersBySearch({ commit }, searchData) {
      try {
        const { data } = await api.fetchPostsUsersBySearch(searchData);
        commit('SET_SEARCH', data);
        return data;
      } catch (error) {
        console.error('getPostsUsersBySearch error:', error);
        throw error;
      }
    },

    async createPost({ commit }, post) {
      try {
        const { data } = await api.createPost(post);
        commit('ADD_POST', data.data);
        return data.data;
      } catch (error) {
        console.error('createPost error:', error);
        throw error;
      }
    },

    async updatePost({ commit, rootGetters }, payload) {
      try {
        const userId = rootGetters['auth/currentUserId'];

        const postData = {
          title: payload.title,
          message: payload.message,
          creator: userId,
          selectedFile: payload.selectedFile,
        };

        const { data } = await api.updatePost(payload.id, postData);
        commit('UPDATE_POST_IN_LIST', data.data);
        commit('SET_POST', data.data);
        return data.data;
      } catch (error) {
        console.error('updatePost error:', error);
        throw error;
      }
    },

    async likePost({ commit }, id) {
      try {
        const { data } = await api.likePost(id);
        commit('UPDATE_POST_IN_LIST', data.data);
        return data.data;
      } catch (error) {
        console.error('likePost error:', error);
        throw error;
      }
    },

    async commentPost({ commit }, form) {
      try {
        const { data } = await api.commentPost(form.value, form.id);
        commit('UPDATE_POST_IN_LIST', data.data);
        return data.data;
      } catch (error) {
        console.error('commentPost error:', error);
        throw error;
      }
    },

    async deleteComment({ commit }, { postId, commentId }) {
      try {
        const { data } = await api.deleteComment(postId, commentId);
        commit('UPDATE_POST_IN_LIST', data.data);
        return data.data;
      } catch (error) {
        console.error('deleteComment error:', error);
        throw error;
      }
    },

    async deletePost({ commit }, id) {
      try {
        await api.deletePost(id);
        commit('REMOVE_POST', id);
      } catch (error) {
        console.error('deletePost error:', error);
        throw error;
      }
    }
  }
};

export default Posts;