// Transform with inline type cast
transform_inline
    from s3
        bucket data
        region us-east-1
        prefix metrics
        format delta

    extract
        json select message
            raw_score raw_score string

    transform
        score = raw_score decimal(18,2)
        | default(0)
        | cast bigint

    select
        score bigint