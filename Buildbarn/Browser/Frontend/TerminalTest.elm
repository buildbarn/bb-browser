module TerminalTest exposing (formattedTextFragments, inputSequence)

import Buildbarn.Browser.Frontend.Terminal as Terminal exposing (Color(..), InputSequence(..), defaultAttributes)
import Expect
import Parser
import Test exposing (Test)


inputSequence : Test
inputSequence =
    [ ( "", Nothing )
    , ( "\u{001B}", Nothing )
    , ( "Hello", Just (TextFragment "Hello") )
    , ( "Hel\u{001B}lo", Just (TextFragment "Hel") )
    , ( "\u{001B}[m", Just (SelectGraphicRendition []) )
    , ( "\u{001B}[XYZm", Nothing )
    , ( "\u{001B}[35;44m", Just (SelectGraphicRendition [ 35, 44 ]) )
    ]
        |> List.map
            (\( input, result ) ->
                Test.test ("Input sequence " ++ input)
                    (\_ ->
                        input
                            |> Parser.run Terminal.inputSequence
                            |> Result.toMaybe
                            |> Expect.equal result
                    )
            )
        |> Test.describe "Shell.inputSequence"


formattedTextFragments : Test
formattedTextFragments =
    Test.describe "Shell.formattedTextFragments"
        [ Test.test "example"
            (\_ ->
                """\u{001B}[1mbla.c:4:10: \u{001B}[0m\u{001B}[0;1;35mwarning: \u{001B}[0m\u{001B}[1mmissing terminating '"' character [-Winvalid-pp-token]\u{001B}[0m
  printf("Hello, world!
);
\u{001B}[0;1;32m         ^
\u{001B}[0m\u{001B}[1mbla.c:4:10: \u{001B}[0m\u{001B}[0;1;31merror: \u{001B}[0m\u{001B}[1mexpected expression\u{001B}[0m
\u{001B}[1mbla.c:5:2: \u{001B}[0m\u{001B}[0;1;31merror: \u{001B}[0m\u{001B}[1mexpected '}'\u{001B}[0m
}
\u{001B}[0;1;32m ^
\u{001B}[0m\u{001B}[1mbla.c:3:12: \u{001B}[0m\u{001B}[0;1;30mnote: \u{001B}[0mto match this '{'\u{001B}[0m
int main() {
\u{001B}[0;1;32m           ^
\u{001B}[0m1 warning and 2 errors generated."""
                    |> Parser.run (Terminal.formattedTextFragments defaultAttributes)
                    |> Expect.equal
                        (Ok
                            { finalAttributes = defaultAttributes
                            , textFragments =
                                [ ( { defaultAttributes | bold = True }, "bla.c:4:10: " )
                                , ( { defaultAttributes | bold = True, foreground = Just Magenta }, "warning: " )
                                , ( { defaultAttributes | bold = True }, "missing terminating '\"' character [-Winvalid-pp-token]" )
                                , ( defaultAttributes, "\n  printf(\"Hello, world!\n);\n" )
                                , ( { defaultAttributes | bold = True, foreground = Just Green }, "         ^\n" )
                                , ( { defaultAttributes | bold = True }, "bla.c:4:10: " )
                                , ( { defaultAttributes | bold = True, foreground = Just Red }, "error: " )
                                , ( { defaultAttributes | bold = True }, "expected expression" )
                                , ( defaultAttributes, "\n" )
                                , ( { defaultAttributes | bold = True }, "bla.c:5:2: " )
                                , ( { defaultAttributes | bold = True, foreground = Just Red }, "error: " )
                                , ( { defaultAttributes | bold = True }, "expected '}'" )
                                , ( defaultAttributes, "\n}\n" )
                                , ( { defaultAttributes | bold = True, foreground = Just Green }, " ^\n" )
                                , ( { defaultAttributes | bold = True }, "bla.c:3:12: " )
                                , ( { defaultAttributes | bold = True, foreground = Just Black }, "note: " )
                                , ( defaultAttributes, "to match this '{'\nint main() {\n" )
                                , ( { defaultAttributes | bold = True, foreground = Just Green }, "           ^\n" )
                                , ( defaultAttributes, "1 warning and 2 errors generated." )
                                ]
                            }
                        )
            )
        ]
