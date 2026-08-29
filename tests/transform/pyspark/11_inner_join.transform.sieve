// Inner join with condition
source_a
    from s3
        bucket db
        region us-east-1
        prefix table_a
        format delta

    extract
        json select message
            id id bigint
            name name string

source_b
    from s3
        bucket db
        region us-east-1
        prefix table_b
        format delta

    extract
        json select message
            id id bigint
            description description string

joined = source_a /\ source_b -> source_a.id = source_b.id
    select
        source_a.id id
        source_a.name name
        source_b.description description