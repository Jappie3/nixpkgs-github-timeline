function loadTimeline() {
  let repo = "nixos/nixpkgs";

  const params = new URLSearchParams(location.search);
  params.set("repo", repo);
  window.history.replaceState(
    {},
    "",
    `${location.pathname}?${params.toString()}`,
  );

  fetch("data/" + repo + ".json")
    .then(function (response) {
      if (response.status !== 200) {
        console.log(
          "Looks like there was a problem. Status Code: " + response.status,
        );
        return;
      }

      response.text().then(function (respBody) {
        let timelineData = JSON.parse(respBody, JSON.dateParser);

        let timeline = timelineData["timeline"].slice(
          0,
          timelineData["timeline"].length - 1,
        );

        populateGraph(timeline, repo);
      });
    })
    .catch(function (err) {
      console.log("Fetch Error", err);
    });
}

function populateGraph(timeline, repo) {
  var issues = {
    type: "scatter",
    name: "Issues",
    x: timeline.map((a) => a.day.replace("T00:00:00Z", " 00:00:00")),
    y: timeline.map((a) => a["open_issues"]),
  };
  var prs = {
    type: "scatter",
    name: "PRs",
    x: timeline.map((a) => a.day.replace("T00:00:00Z", " 00:00:00")),
    y: timeline.map((a) => a["open_prs"]),
  };

  let layout = {
    title: {
      text: '<a href="https://github.com/' + repo + '">' + repo + "</a>",
    },
    showSendToCloud: false,
    autosize: true,
  };
  var data = [issues, prs];
  Plotly.newPlot("graph", data, layout, { displayModeBar: false });
}

document.addEventListener("DOMContentLoaded", function () {
  loadTimeline();
});
