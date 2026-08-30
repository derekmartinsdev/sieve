// JSON extract from derived column
source_with_array
    from s3
        bucket data
        region us-east-1
        prefix raw
        format delta

    extract
        json select message
            id id bigint
            tags tags string

extracted
    from source_with_array

    extract
        json extract tags array
            tag tag string