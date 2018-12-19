/*
Copyright 2015 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
import 'whatwg-fetch';
import localStorage from './localStorage';
import cfg from './../config';

const defaultCfg = {
  credentials: 'same-origin',
  headers: {
    'Accept': 'application/json',
    'Content-Type': 'application/json; charset=utf-8'
  }
}

const api = {

  get(url){
    return api.ajax(url);
  },

  post(url, data){
    return api.ajax(url, {
      body: JSON.stringify(data),
      method: 'POST'
    });
  },

  delete(url, data){
    return api.ajax(url, {
      body: JSON.stringify(data),
      method: 'DELETE'
    });
  },

  put(url, data){
    return api.ajax(url, {
      body: JSON.stringify(data),
      method: 'PUT'
    });
  },

  ajax(url, params) {
    url = cfg.baseUrl + url;
    const options = {
      ...defaultCfg,
      ...params
    };

    options.headers = {
      ...options.headers,
      ...getAuthHeaders()
    }

    return new Promise((resolve, reject) => {
      fetch(url, options)
        .then(parseJSON)
        .then(response => {
          if (response.ok) {
            return resolve(response.json);
          }

          const err = new Error(getErrorText(response.json));
          err.status = response.status;
          return reject(err);
        })
        .catch(err => {
          reject(err);
        })
    });
  }
}

function parseJSON(response) {
  return new Promise((resolve, reject) => {
    return response
      .json()
      .then(json => resolve({status: response.status, ok: response.ok, json}))
      .catch(err => reject(err))
  });
}

export function getAuthHeaders() {
  const accessToken = getAccessToken();
  const csrfToken = getXCSRFToken();
  return {
    'X-CSRF-Token': csrfToken,
    'Authorization': `Bearer ${accessToken}`
  }
}

export function getNoCacheHeaders() {
  return {
    'cache-control': 'max-age=0',
    'expires': '0',
    'pragma': 'no-cache'
  }
}

export const getXCSRFToken = () => {
  const metaTag = document.querySelector('[name=grv_csrf_token]');
  return metaTag ? metaTag.content : '';
}

export function getAccessToken() {
  const bearerToken = localStorage.getBearerToken() || {};
  return bearerToken.accessToken;
}

function getErrorText(json){
  const msg = 'Unknown error';

  if(json && json.error){
    return json.error.message || msg;
  }

  if(json && json.message){
    return json.message;
  }

  if (json && json.error) {
    return json.error.message || msg;
  }

  if (json.responseText) {
    return json.responseText;
  }

  return msg;
}

export default api;
