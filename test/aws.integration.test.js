/* eslint-disable max-len */
const expect = require('chai').expect;
const {AWS, uploadFileToS3, getFileFromS3} = require('../lib/aws');
const config = require('../config');
const {parseSchedule, compareSchedules, serializeSchedule, deserializeSchedule} = require('../lib/helper_functions');

describe(`AWS Integration Tests`, function() {
  it(`confirms that the environment variables for connecting to AWS are set`, function() {
    expect(process.env.AWS_ACCESS_KEY_ID).to.be.a('string');
    expect(process.env.AWS_SECRET_ACCESS_KEY).to.be.a('string');
    expect(process.env.AWS_DEFAULT_REGION).to.be.a('string');
    expect(process.env.AWS_ACCESS_KEY_ID).to.equal(config.aws_access_token_id);
    expect(process.env.AWS_SECRET_ACCESS_KEY).to.equal(config.aws_access_token_secret);
    expect(process.env.AWS_DEFAULT_REGION).to.equal(config.aws_default_region);
  });

  it(`can list the buckets (i.e. validating credentials and permissions)`, async function() {
    // Create S3 service object
    const s3 = new AWS.S3({apiVersion: '2006-03-01'});

    // Call S3 to list the buckets
    const result = await s3.listBuckets().promise();
    expect(result.Buckets).to.be.an('array');
    expect(result.Buckets.length > 0).to.be.true;
  });

  it(`can upload a string to a file in AWS S3`, async function() {
    const result = await uploadFileToS3('Test Contents', 'testfile.txt');
    expect(result.Bucket).to.equal(config.aws_s3_bucket);
    expect(result.Key).to.equal('testfile.txt');
  });

  it(`can read from a file in AWS S3`, async function() {
    const result = await getFileFromS3('testfile.txt');
    expect(result.toString('utf-8')).to.equal('Test Contents');
  });

  it(`can see the file created in previous test`, async function() {
    // Create S3 service object
    const s3 = new AWS.S3({apiVersion: '2006-03-01'});

    // First check if the file exists using `headObject`
    const result = await s3.headObject({
      Bucket: config.aws_s3_bucket,
      Key: 'testfile.txt',
    }).promise();
    expect(!result.err).to.be.true; // any error indicates an issue in locating the file/object
  });

  it(`can serialize and deserialize the schedule to disk`, async function() {
    const input = 'Upcoming Schedule\n\nWear baseball pants or sweatpants to every practice, and bring all of your baseball gear.\n\n​\n\nTUESDAY, 10/3\n\nPractice, Warren, 4:45–6:45\n\nLate: Aiden, Sam, Zach\n\nOut: Matty\n\n​\n\nTHURSDAY, 10/5\n\nPractice, Warren, 4:45–6:45\n\nLate: —\n\nOut: Matty\n\n​\n\nSATURDAY, 10/7\n\nPractice, Warren, 3:00–5:30\n\nLate: —\n\nOut: Connor\n\n​\n\nSUNDAY, 10/8\n\nPractice, Warren, 3:00–5:30\n\nLate: —\n\nOut: Connor\n\n  \n\nSchedule by Season\n\nOur tentative plan for the months ahead. More details to come. \n\n \n\nFALL 20…Th/Sa/Su), September–November, at Warren Field (starting 9/5).\n\n \n\nWINTER 2024\n\nIndoor practices on Saturday or Sunday evenings, January–March, at Brookline HS Tappan Pavilion.\n\n \n\nSPRING 2024\n\n​Doubleheaders on Saturdays, April–June, in the 12U Division of the Select League.\n\n​\n\n\n\nPractices once or twice per week (TBD).\n\n\nPlayoffs in July, if we qualify.\n\n\nSchedule will not conflict with BYB Majors.\n\n\nPlaying time will depend on baseball skills, commitment, focus, work ethic, and attitude.​\n\n';
    const a = parseSchedule(input);
    await serializeSchedule(a, 'serializedTestSchedule.bson');
    const b = await deserializeSchedule('serializedTestSchedule.bson');
    const result = compareSchedules(a, b);
    expect(result['unchanged'].size).to.equal(4); // all entries unchanged
    expect(result['added'].size).to.equal(0);
    expect(result['deleted'].size).to.equal(0);
    expect(result['modified'].size).to.equal(0);
  });

  after(async function() {
    // Create S3 service object
    const s3 = new AWS.S3({apiVersion: '2006-03-01'});

    try {
      // Call S3 to delete the test file.
      await s3.deleteObject({
        Bucket: config.aws_s3_bucket,
        Key: 'testfile.txt',
      }).promise();
    } catch (e) {
      console.error(e);
    }

    try {
      // Call S3 to delete the test file.
      await s3.deleteObject({
        Bucket: config.aws_s3_bucket,
        Key: 'serializedTestSchedule.bson',
      }).promise();
    } catch (e) {
      console.error(e);
    }
  });
});

