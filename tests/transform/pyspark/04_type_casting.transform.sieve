// JSON select with full type casting
type_casting
    from s3
        bucket finance
        region sa-east-1
        prefix transactions
        format delta

    extract
        json select message
            id id bigint
            amount amount decimal(18,4)
            rate rate decimal(10,6)
            settled_date settled_date date(YYYY-MM-DD)
            description description string
            active active bigint