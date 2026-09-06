// Transform chain with all operations
transform_chain
    from s3
        bucket finance
        region us-east-1
        prefix payments
        format delta

    extract
        json select message
            raw_amount raw_amount string

    transform
        clean_amount = raw_amount
        | cast decimal(18,2)
        | default(0) decimal(18,2)
        | cast string
        | replace(".", ",")
        | prefix("R$")
        | hash()

    select
        clean_amount string