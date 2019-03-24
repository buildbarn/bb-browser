module Buildbarn.Browser.Frontend.Terminal exposing
    ( Attributes
    , Color(..)
    , FormattedTextFragment
    , FormattedTextFragments
    , InputSequence(..)
    , defaultAttributes
    , formattedTextFragments
    , inputSequence
    )

{-| A series of parsers that can be used to convert VT100/ECMA terminal
output to a list of text fragments that can be rendered with graphical
attributes applied. These parsers may be used to render command output
on a web page.
-}

import Html exposing (Html)
import Parser exposing (Parser)


{-| Parses a sequence of characters that don't contain any terminal
escape sequences.
-}
textFragment : Parser String
textFragment =
    (\c -> c /= '\u{001B}')
        |> Parser.chompWhile
        |> Parser.getChompedString
        |> Parser.andThen
            (\s ->
                if String.isEmpty s then
                    Parser.problem "text fragment must be non-empty"

                else
                    Parser.succeed s
            )


{-| Parses a single Select Graphic Rendition (SGR) escape sequence,
returning any provided parameters as a list of integers.
-}
selectGraphicRendition : Parser (List Int)
selectGraphicRendition =
    Parser.sequence
        { start = "\u{001B}["
        , separator = ";"
        , end = "m"
        , spaces = Parser.succeed ()
        , item = Parser.int
        , trailing = Parser.Forbidden
        }


type InputSequence
    = TextFragment String
    | SelectGraphicRendition (List Int)


{-| Parses a single chunk of text or escape sequence at the start of an
input sequence.
-}
inputSequence : Parser InputSequence
inputSequence =
    Parser.oneOf
        [ Parser.map TextFragment textFragment
        , Parser.map SelectGraphicRendition selectGraphicRendition
        ]


{-| Set of eight base colors supported by 16-color terminals. The eight
highlighted versions of these colors are normally only picked when the
text to be displayed is bold.
-}
type Color
    = Black
    | Red
    | Green
    | Brown
    | Blue
    | Magenta
    | Cyan
    | White


{-| Visual attributes that are applied to text rendered on a terminal.
-}
type alias Attributes =
    { bold : Bool
    , underline : Bool
    , blink : Bool
    , reverse : Bool
    , foreground : Maybe Color
    , background : Maybe Color
    }


{-| Attributes that should be applied to text when the terminal is in
the initial state.
-}
defaultAttributes : Attributes
defaultAttributes =
    { bold = False
    , underline = False
    , blink = False
    , reverse = False
    , foreground = Nothing
    , background = Nothing
    }


{-| Applies a single SGR parameter to an existing set of terminal
attributes.
-}
applyAttribute : Int -> Attributes -> Attributes
applyAttribute code attributes =
    case code of
        0 ->
            defaultAttributes

        1 ->
            { attributes | bold = True }

        4 ->
            { attributes | underline = True }

        5 ->
            { attributes | blink = True }

        7 ->
            { attributes | reverse = True }

        22 ->
            { attributes | bold = False }

        24 ->
            { attributes | underline = False }

        25 ->
            { attributes | blink = False }

        27 ->
            { attributes | reverse = False }

        30 ->
            { attributes | foreground = Just Black }

        31 ->
            { attributes | foreground = Just Red }

        32 ->
            { attributes | foreground = Just Green }

        33 ->
            { attributes | foreground = Just Brown }

        34 ->
            { attributes | foreground = Just Blue }

        35 ->
            { attributes | foreground = Just Magenta }

        36 ->
            { attributes | foreground = Just Cyan }

        37 ->
            { attributes | foreground = Just White }

        39 ->
            { attributes | foreground = Nothing }

        40 ->
            { attributes | background = Just Black }

        41 ->
            { attributes | background = Just Red }

        42 ->
            { attributes | background = Just Green }

        43 ->
            { attributes | background = Just Brown }

        44 ->
            { attributes | background = Just Blue }

        45 ->
            { attributes | background = Just Magenta }

        46 ->
            { attributes | background = Just Cyan }

        47 ->
            { attributes | background = Just White }

        49 ->
            { attributes | background = Nothing }

        _ ->
            -- Unknown attribute code (e.g., 256 colors). Skip these for now.
            attributes


type alias FormattedTextFragment =
    ( Attributes, String )


type alias FormattedTextFragments =
    { textFragments : List FormattedTextFragment
    , finalAttributes : Attributes
    }


{-| Stores a single text fragment observed in the terminal stream along
with the graphical attributes that currently apply, or changes the
terminal's current attributes according to observed Select Graphic
Rendition escape sequences.
-}
applyInputSequence : FormattedTextFragments -> InputSequence -> FormattedTextFragments
applyInputSequence state sequence =
    case sequence of
        TextFragment newText ->
            { state
                | textFragments =
                    case state.textFragments of
                        ( oldAttributes, oldText ) :: remainingTextFragments ->
                            if state.finalAttributes == oldAttributes then
                                -- Successive text fragments with identical attributes. Merge them.
                                ( state.finalAttributes, oldText ++ newText ) :: remainingTextFragments

                            else
                                -- Successive text fragments with different attributes.
                                ( state.finalAttributes, newText ) :: state.textFragments

                        _ ->
                            -- Initial text fragment.
                            ( state.finalAttributes, newText ) :: state.textFragments
            }

        SelectGraphicRendition codes ->
            if List.isEmpty codes then
                -- The "^[[m" escape sequence should reset the default attributes.
                { state | finalAttributes = defaultAttributes }

            else
                -- Apply all provided attributes.
                { state | finalAttributes = List.foldl applyAttribute state.finalAttributes codes }


{-| Parses a string terminal application output containing escape
sequences into a list of text fragments, having graphical attributes.
-}
formattedTextFragments : Attributes -> Parser FormattedTextFragments
formattedTextFragments initialAttributes =
    Parser.loop
        { textFragments = [], finalAttributes = initialAttributes }
    <|
        \state ->
            Parser.oneOf
                [ inputSequence
                    |> Parser.map (\s -> Parser.Loop (applyInputSequence state s))
                , Parser.succeed ()
                    |> Parser.map
                        (\_ ->
                            Parser.Done
                                -- applyInputSequence prepends text fragments, instead of appending them.
                                { state
                                    | textFragments = List.reverse state.textFragments
                                }
                        )
                ]
