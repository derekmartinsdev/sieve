// Left join with parenthesized condition
left_table
    from s3
        bucket db
        region us-east-1
        prefix customers
        format delta

    extract
        json select message
            id id bigint
            name name string

right_table
    from s3
        bucket db
        region us-east-1
        prefix orders
        format delta

    extract
        json select message
            customer_id customer_id bigint
            total total decimal(18,2)

left_joined = left_table (c) /\ right_table (o) -> (
    c.id = o.customer_id
), left
    select
        c.id customer_id
        c.name name
        o.total total