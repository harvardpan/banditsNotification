const expect = require('chai').expect;
const {retrieveApiToken, retrieveSecret} = require('../lib/vault');

describe(`HCP Vault Secrets Tests`, function() {
  it(`confirms that we can retrieve the API token`, async function() {
    const result = await retrieveApiToken();
    expect(result).to.be.a('string');
    expect(result.length > 0).to.be.true;
  });

  it(`can retrieve a HCP Vault Secret secret`, async function() {
    const apiToken = await retrieveApiToken();
    const secret = await retrieveSecret(apiToken, 'AWS_ACCESS_KEY_ID');
    expect(secret).to.be.a('string');
    expect(secret.length > 0).to.be.true;
  });
});
