require('dotenv').config()

// Defines the Twitter configuration
const config = {
    consumer_key: process.env.TWITTER_CONSUMER_KEY, // API Key
    consumer_secret: process.env.TWITTER_CONSUMER_SECRET, // API Key Secret
    access_token_key: process.env.TWITTER_ACCESS_TOKEN_KEY,
    access_token_secret: process.env.TWITTER_ACCESS_TOKEN_SECRET,
    runInterval: parseInt(process.env.RUN_INTERVAL), // # of seconds between checks/runs
    twitterUserHandle: process.env.TWITTER_USER_HANDLE, // used primarily for testing connectivity
};

module.exports = config;
