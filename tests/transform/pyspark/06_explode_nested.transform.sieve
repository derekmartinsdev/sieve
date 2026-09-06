// JSON explode with nested JSON fields inside the array
explode_nested
    from s3
        bucket logs
        region us-east-1
        prefix api_logs
        format delta

    extract
        json explode message.records array
            request_id request_id string
            request.path path string
            request.method method string
            response.status status bigint
            response.duration_ms duration bigint