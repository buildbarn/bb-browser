module ShellTest exposing (quote)

import Buildbarn.Browser.Frontend.Shell as Shell
import Expect
import Test



-- Test vectors obtained from:
-- https://github.com/kballard/go-shellquote/pull/7


quote : Test.Test
quote =
    [ ( "test", "test" )
    , ( "hello goodbye", "'hello goodbye'" )
    , ( "hello", "hello" )
    , ( "goodbye", "goodbye" )
    , ( "don't you know the dewey decimal system?", "'don'\\''t you know the dewey decimal system?'" )
    , ( "don't", "don\\'t" )
    , ( "you", "you" )
    , ( "know", "know" )
    , ( "the", "the" )
    , ( "dewey", "dewey" )
    , ( "decimal", "decimal" )
    , ( "system?", "system\\?" )
    , ( "~user", "\\~user" )
    , ( "u~ser", "u~ser" )
    , ( " ~user", "' ~user'" )
    , ( "!~user", "\\!~user" )
    , ( "foo*", "foo\\*" )
    , ( "M{ovies,usic}", "M\\{ovies,usic}" )
    , ( "ab[cd]", "ab\\[cd]" )
    , ( "%3", "%3" )
    , ( "one", "one" )
    , ( "", "''" )
    , ( "three", "three" )
    , ( "some(parentheses)", "some\\(parentheses\\)" )
    , ( "$some_ot~her_)spe!cial_*_characters", "\\$some_ot~her_\\)spe\\!cial_\\*_characters" )
    , ( "tabs\tand", "'tabs\tand'" )
    , ( "spaces and", "'spaces and'" )
    , ( "newlines\n", "'newlines\n'" )
    , ( "-\noh my!", "'-\noh my!'" )
    , ( "⋃ₙⅰ𝄴𝕺Ⅾ€", "⋃ₙⅰ𝄴𝕺Ⅾ€" )
    , ( "⋃ ₙ ⅰ 𝄴 𝕺 Ⅾ €", "'⋃ ₙ ⅰ 𝄴 𝕺 Ⅾ €'" )
    , ( "$", "\\$" )
    , ( "~", "\\~" )
    , ( "~$", "\\~\\$" )
    , ( "$~", "\\$~" )
    , ( "'quoted'", "\\'quoted\\'" )
    , ( "\"quoted\"", "\\\"quoted\\\"" )
    , ( "\"quoted with spaces\"", "'\"quoted with spaces\"'" )
    , ( "'", "\\'" )
    , ( "'''", "\\'\\'\\'" )
    , ( "' ", "\\'' '" )
    , ( "''' ", "\\'\\'\\'' '" )
    , ( " '", "' '\\'" )
    , ( " '''", "' '\\'\\'\\'" )
    ]
        |> List.indexedMap
            (\index ->
                \( before, after ) ->
                    Test.test ("test vector " ++ String.fromInt index)
                        (\_ -> Expect.equal after (Shell.quote before))
            )
        |> Test.describe "Shell.quote"
