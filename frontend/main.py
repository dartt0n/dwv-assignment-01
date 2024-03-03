from dash import Dash, html, dcc, dash_table
import plotly.express as px
import plotly.graph_objects as go
from pymongo import MongoClient
import pandas as pd
from collections import Counter

# Initialize Dash app
app = Dash(__name__)

# Monokai color palette
MONOKAI = {
    "background": "#272822",
    "text": "#F8F8F2",
    "purple": "#AE81FF",
    "green": "#A6E22E",
    "orange": "#FD971F",
    "yellow": "#E6DB74",
    "pink": "#F92672",
    "blue": "#66D9EF",
    "gray": "#75715E",
}

template = {
    "layout": {
        "plot_bgcolor": MONOKAI["background"],
        "paper_bgcolor": MONOKAI["background"],
        "font": {"color": MONOKAI["text"]},
        "title": {"font": {"color": MONOKAI["blue"]}},
        "xaxis": {"gridcolor": MONOKAI["gray"], "color": MONOKAI["text"]},
        "yaxis": {"gridcolor": MONOKAI["gray"], "color": MONOKAI["text"]},
    }
}
# Connect to MongoDB
client = MongoClient("mongodb://replace-admin:replace-password@mongodb:27017/")
db = client["films"]
collection = db["films"]

data = list(collection.find())
df = pd.DataFrame(data)


# Data processing
def process_data():
    # Films per year
    films_per_year = df["releaseYear"].value_counts().sort_index()

    # Box office per year
    box_office_per_year = df.groupby("releaseYear")["boxOffice"].sum()

    # Worldwide gross per year
    df["worldwideGross"] = df["worldwideGross"].astype(float)
    gross_per_year = df.groupby("releaseYear")["worldwideGross"].sum()

    # Top directors
    all_directors = [
        director for directors in df["directors"] for director in directors
    ]
    top_directors = Counter(all_directors).most_common(3)

    # Top countries
    top_countries = df["countryOfOrigin"].value_counts().head()

    return (
        films_per_year,
        box_office_per_year,
        gross_per_year,
        top_directors,
        top_countries,
    )


# Process data
films_per_year, box_office_per_year, gross_per_year, top_directors, top_countries = (
    process_data()
)

# Create figures with Monokai theme
# Films per year figure
fig1 = px.bar(
    x=films_per_year.index,
    y=films_per_year.values,
    title="Films Released per Year",
    labels={"x": "Year", "y": "Number of Films"},
    template=template,
)
fig1.update_traces(marker_color=MONOKAI["purple"])

# Box office per year figure
fig2 = px.line(
    x=box_office_per_year.index,
    y=box_office_per_year.values,
    title="Box Office Revenue per Year",
    labels={"x": "Year", "y": "Revenue ($)"},
    template=template,
)
fig2.update_traces(line_color=MONOKAI["green"])

# Worldwide gross per year figure
fig3 = px.area(
    x=gross_per_year.index,
    y=gross_per_year.values,
    title="Worldwide Gross Revenue per Year",
    labels={"x": "Year", "y": "Revenue ($)"},
    template=template,
)
fig3.update_traces(fill="tonexty", line_color=MONOKAI["orange"])

# Top directors figure
fig4 = px.bar(
    x=[d[0] for d in top_directors],
    y=[d[1] for d in top_directors],
    title="Top 3 Directors by Number of Films",
    labels={"x": "Director", "y": "Number of Films"},
    template=template,
)
fig4.update_traces(marker_color=MONOKAI["pink"])

# Top countries figure
fig5 = px.pie(
    values=top_countries.values,
    names=top_countries.index,
    title="Top Film Producing Countries",
    template=template,
)
fig5.update_traces(
    marker_colors=[
        MONOKAI["purple"],
        MONOKAI["green"],
        MONOKAI["orange"],
        MONOKAI["pink"],
        MONOKAI["blue"],
    ]
)

# Create layout
app.layout = html.Div(
    [
        html.H1(
            "Movie Industry Analysis Dashboard",
            style={"textAlign": "center", "color": MONOKAI["blue"], "marginBottom": 30},
        ),
        html.H3(
            [
                html.Div(
                    "Source code: ",
                    style={
                        "textAlign": "center",
                        "color": MONOKAI["blue"],
                    },
                ),
                html.A(
                    "dwv-assignment-01",
                    href="https://github.com/dartt0n/dwv-assignment-01",
                    target="_blank",
                    style={"color": MONOKAI["blue"], "marginBottom": 30},
                ),
            ]
        ),
        # Films per year
        html.Div(
            [
                html.H2(
                    "Number of Films Released per Year",
                    style={"color": MONOKAI["blue"]},
                ),
                html.P(
                    "This graph shows the distribution of films released each year, helping us understand industry production trends.",
                    style={"color": MONOKAI["text"]},
                ),
                dcc.Graph(figure=fig1),
            ],
            style={"marginBottom": 40},
        ),
        # Box office per year
        html.Div(
            [
                html.H2(
                    "Total Box Office Revenue per Year",
                    style={"color": MONOKAI["blue"]},
                ),
                html.P(
                    "This visualization displays the annual box office revenue, indicating overall market performance.",
                    style={"color": MONOKAI["text"]},
                ),
                dcc.Graph(figure=fig2),
            ],
            style={"marginBottom": 40},
        ),
        # Worldwide gross per year
        html.Div(
            [
                html.H2(
                    "Worldwide Gross Revenue per Year", style={"color": MONOKAI["blue"]}
                ),
                html.P(
                    "This chart shows the global revenue trends, highlighting the international success of films.",
                    style={"color": MONOKAI["text"]},
                ),
                dcc.Graph(figure=fig3),
            ],
            style={"marginBottom": 40},
        ),
        # Top directors
        html.Div(
            [
                html.H2(
                    "Top 3 Most Popular Directors", style={"color": MONOKAI["blue"]}
                ),
                html.P(
                    "These are the directors with the most films in the database, showcasing industry leaders.",
                    style={"color": MONOKAI["text"]},
                ),
                dcc.Graph(figure=fig4),
            ],
            style={"marginBottom": 40},
        ),
        # Top countries
        html.Div(
            [
                html.H2(
                    "Top Film Producing Countries", style={"color": MONOKAI["blue"]}
                ),
                html.P(
                    "This graph shows which countries are leading in film production.",
                    style={"color": MONOKAI["text"]},
                ),
                dcc.Graph(figure=fig5),
            ],
            style={"marginBottom": 40},
        ),
    ],
    style={
        "padding": "20px",
        "backgroundColor": MONOKAI["background"],
        "minHeight": "100vh",
        "display": "flex",
        "alignItems": "center",
        "justifyContent": "center",
        "flexDirection": "column",
        "fontFamily": "Lora",
    },
)

if __name__ == "__main__":
    app.run_server(debug=False)
