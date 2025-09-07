/* eslint-disable max-len */
/* eslint-disable require-jsdoc */
'use strict';

const {TwitterApi} = require('twitter-api-v2');

async function testTwitterCredentials() {
  console.log('🧪 Testing Twitter credentials with Node.js (replicating original tweetScreenshot)...');
  
  try {
    // Load config the same way the Go test does
    const {exec} = require('child_process');
    const {promisify} = require('util');
    const execAsync = promisify(exec);
    
    // Decrypt the config file using sops
    console.log('📄 Decrypting test config...');
    const {stdout} = await execAsync('sops -d test_config.yaml');
    const yaml = require('js-yaml');
    const cfg = yaml.load(stdout);
    
    if (!cfg.app || !cfg.app.urls || cfg.app.urls.length === 0) {
      throw new Error('No URLs configured in test config');
    }
    
    // Use first URL config (same as Go test)
    const twitterConfig = cfg.app.urls[0].twitter;
    console.log(`🔑 Using credentials for: @${twitterConfig.user_handle}`);
    
    // Create Twitter client (same as original tweetScreenshot function)
    const client = new TwitterApi({
      appKey: twitterConfig.consumer_key,
      appSecret: twitterConfig.consumer_secret,
      accessToken: twitterConfig.access_token,
      accessSecret: twitterConfig.access_token_secret,
    });
    
    // Test 1: Verify credentials
    console.log('🔍 Testing credential verification...');
    const user = await client.v1.verifyCredentials();
    console.log(`✅ Credentials verified for @${user.screen_name}`);
    
    // Test 2: Upload media using v1 API (exactly like original tweetScreenshot)
    console.log('📸 Testing media upload (v1 API)...');
    
    // Create a proper PNG using sharp (like Puppeteer would create)
    const sharp = require('sharp');
    const testImageData = await sharp({
      create: {
        width: 100,
        height: 100,
        channels: 4,
        background: { r: 0, g: 100, b: 200, alpha: 1 }
      }
    })
    .png()
    .toBuffer();
    
    // Upload media using v1 API (exactly like original code)
    const mediaIds = await Promise.all([
      client.v1.uploadMedia(testImageData, {
        type: 'png', // Use original parameter format
      }),
    ]);
    console.log(`✅ Media uploaded successfully! ID: ${mediaIds[0]}`);
    
    // Test 3: Post tweet with media using v2 API (exactly like original tweetScreenshot)
    console.log('📝 Testing tweet with image posting (v2 API)...');
    const testMessage = `🧪 Node.js credential test with image - ${new Date().toISOString().slice(0, 19).replace('T', ' ')}. https://github.com/test/integration`;
    
    // Post tweet with media using v2 API (exactly like original)
    const tweet = await client.v2.tweet({
      text: testMessage,
      media: {media_ids: mediaIds},
    });
    
    console.log(`✅ Tweet posted successfully! ID: ${tweet.data.id}`);
    console.log(`📱 Tweet text: ${testMessage}`);
    
    // Clean up - delete the test tweet using v1 API
    console.log('🧹 Cleaning up test tweet...');
    await client.v1.deleteTweet(tweet.data.id);
    console.log('✅ Test tweet deleted successfully');
    
  } catch (error) {
    console.error('💥 Test failed:', error.message);
    if (error.data) {
      console.error('📊 Error data:', JSON.stringify(error.data, null, 2));
    }
    if (error.errors) {
      console.error('📊 Errors:', JSON.stringify(error.errors, null, 2));
    }
    throw error;
  }
}

// Run the test
testTwitterCredentials()
  .then(() => {
    console.log('🎉 All tests passed! Credentials work exactly like original tweetScreenshot function.');
    process.exit(0);
  })
  .catch((error) => {
    console.error('🚨 Test suite failed:', error.message);
    process.exit(1);
  });