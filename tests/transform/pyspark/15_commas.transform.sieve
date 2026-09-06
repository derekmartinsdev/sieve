// Select with commas between fields
commas
    from s3
        bucket data
        region us-east-1
        prefix users
        format delta

    extract
        json select message
            id , id bigint
            name , name string
            email , email string
            age , age bigint