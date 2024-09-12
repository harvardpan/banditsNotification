/* eslint-disable max-len */
const axios = require('axios');
const config = require('../config');

/**
 * Retrieves the HCP API Token, by providing it with the Client ID
 * Secret.
 *
 * @async
 * @return {String} the API token to be used for additional API calls
 */
async function retrieveApiToken() {
  try {
    const result = await axios.post('/oauth/token', {
      audience: 'https://api.hashicorp.cloud',
      grant_type: 'client_credentials',
      client_id: `${config.hcp_client_id}`,
      client_secret: `${config.hcp_client_secret}`,
    },
    {
      baseURL: 'https://auth.hashicorp.com',
      headers: {'content-type': 'application/json'},
    });
    if (result.status !== 200) {
      return null;
    }
    return result.data.access_token;
  } catch (e) {
    console.error(e);
  }
  return null;
}

/**
 * Makes a request against HCP Vault Secrets to retrieve a particular secret.
 * This will query using the API Token gotten from `retrieveApiToken()`
 *
 * @async
 * @param {String} apiToken the API Token gotten using `retrieveApiToken()`
 * @param {String} secretName the key of the secret to retrieve
 * @return {String} the value stored for the secret
 */
async function retrieveSecret(apiToken, secretName, hcp_app_name = '') {
  try {
    if (secretName.startsWith('AWS_')) {
      // We override the app name with the AWS app name
      hcp_app_name = 'AWS-S3-Access';
    }
    const result = await axios.get(`secrets/2023-06-13/organizations/${config.hcp_organization_id}/projects/${config.hcp_project_id}/apps/${hcp_app_name}/open/${secretName}`,
        {
          baseURL: 'https://api.cloud.hashicorp.com',
          headers: {
            'content-type': 'application/json',
            'Authorization': `Bearer ${apiToken}`,
          },
        });
    if (result.status !== 200) {
      return null;
    }
    return result.data.secret.version.value;
  } catch (e) {
    console.error(e);
  }
  return null;
}

module.exports = {
  retrieveApiToken,
  retrieveSecret,
};
