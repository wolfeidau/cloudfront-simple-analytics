# cloudfront-simple-analytics

This project illustrates how to setup a simple analytics service for CloudFront. It uses lambda, kinesis firehose, and Athena to enable you to capture page views using a novel strategy described in [How Bear does analytics with CSS](https://herman.bearblog.dev/how-bear-does-analytics-with-css/).

The primary goal of this project was to explore a few different options for analytics, and to learn more about CloudFront, Lambda, and Kinesis Firehose.

# Solutions

This includes a few solutions I wanted to dig into:

* Cloudfront with Lambda Function URL integration, based on [Amazon CloudFront now supports Origin Access Control (OAC) for Lambda function URL origins](https://aws.amazon.com/about-aws/whats-new/2024/04/amazon-cloudfront-oac-lambda-function-url-origins/).
* Kinesis Firehose with Lambda Transform and Dynamic Partitioning [Dynamic Partitioning in Amazon Data Firehose](https://docs.aws.amazon.com/firehose/latest/dev/dynamic-partitioning.html).

# License

This application is released under Apache 2.0 license and is copyright [Mark Wolfe](https://www.wolfe.id.au/?utm_source=cloudfront-simple-analytics).
