// Select with alias only (no type)
select_alias_only
    from s3
        bucket data
        region us-east-1
        prefix records
        format delta

    extract
        json select message
            first_name first_name
            last_name last_name
            birth_date birth_date

    select
        first_name
        last_name
        birth_date