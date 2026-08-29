// Simple JSON select with single column
simple_select
    from s3
        bucket test_bucket
        region us-east-1
        prefix raw_data
        format delta

    extract
        json select message
            name name string