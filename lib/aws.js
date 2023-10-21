const config = require('../config');
// Load the SDK for JavaScript
const AWS = require('aws-sdk');
AWS.config.update({region: config.aws_default_region}); // Set the Region
AWS.config.logger = console; // log API calls to the console
const {Readable} = require('stream');

/**
 * Uploads the text into S3 with the specified filename.
 *
 * @async
 * @param {*} contents The contents of the file that will be uploaded.
 * @param {String} filename the actual filename that should be uploaded.
 * @return {Object} Object with `Location`, `ETag`, `Bucket`, and `Key`
 */
async function uploadFileToS3(contents, filename) {
  // Create S3 service object
  const s3 = new AWS.S3({apiVersion: '2006-03-01'});

  const readableStream = Readable.from(contents);
  // Configure the upload parameters
  const uploadParams = {
    Bucket: config.aws_s3_bucket,
    Key: filename,
    Body: readableStream,
  };

  let data = null;
  try {
    // call S3 to upload file to specified bucket
    data = await s3.upload(uploadParams).promise();
  } catch (e) {
    console.error(e);
  }
  return data;
}

/**
 * Retrieves the contents of an S3 object using `getObject`
 *
 * @async
 * @param {String} filename `Key` for the S3 object to retrieve
 * @return {*} contents of the file
 */
async function getFileFromS3(filename) {
  // Create S3 service object
  const s3 = new AWS.S3({apiVersion: '2006-03-01'});

  // Configure the parameters
  const params = {
    Bucket: config.aws_s3_bucket,
    Key: filename,
  };

  let data = null;
  try {
    // call S3 to upload file to specified bucket
    data = await s3.getObject(params).promise();
  } catch (e) {
    if (e.code !== 'NoSuchKey') {
      // Only log if it's actually something we need to worry about.
      console.error(e);
    }
    return null;
  }
  return data.Body;
}

module.exports = {
  uploadFileToS3,
  getFileFromS3,
  AWS, // export the entire AWS file so it can be re-used
};
