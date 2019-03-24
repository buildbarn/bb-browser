module Buildbarn.Browser.Frontend.Shell exposing (quote)

{-| A library for escaping strings to be pasted in a UNIX shell.
-}


{-| Single-quotes a string, escaping any quotes contained within.
-}
fullyQuoteMiddle : String -> String
fullyQuoteMiddle s =
    "'" ++ String.replace "'" "'\\''" s ++ "'"


{-| Escapes trailing quotes in a string, followed by quoting the rest.
-}
fullyQuoteRight : String -> String
fullyQuoteRight s =
    if String.endsWith "'" s then
        fullyQuoteRight (String.dropRight 1 s) ++ "\\'"

    else
        fullyQuoteMiddle s


{-| Escapes leading quotes in a string, followed by quoting the rest.
-}
fullyQuoteLeft : String -> String
fullyQuoteLeft s =
    case String.uncons s of
        Just ( '\'', rest ) ->
            "\\'" ++ fullyQuoteLeft rest

        _ ->
            fullyQuoteRight s


{-| Escape all characters in a string.
-}
escapeRest : String -> String
escapeRest s =
    s
        |> String.toList
        |> List.concatMap
            (\c ->
                if String.contains (String.fromChar c) "\\'\"`${[|&;<>()*?!" then
                    [ '\\', c ]

                else
                    [ c ]
            )
        |> String.fromList


{-| Escapes leading ~ characters, followed by escaping the remainder.
-}
escapeLeading : String -> String
escapeLeading s =
    case String.uncons s of
        Just ( '~', rest ) ->
            "\\~" ++ escapeRest rest

        _ ->
            escapeRest s


{-| Escapes or quotes a string, making it safe to paste in a UNIX shell.
-}
quote : String -> String
quote s =
    if String.isEmpty s then
        -- Corner case: empty string needs to be quoted.
        "''"

    else if String.any (\c -> c == ' ' || c == '\t' || c == '\n') s then
        -- Quote strings that contain whitespace characters.
        fullyQuoteLeft s

    else
        -- Escape individual characters.
        escapeLeading s
