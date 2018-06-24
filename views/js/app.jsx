const LightsArray = props => {
  console.log(props.lights);
  const lightsItem = props.lights.map(light => {
    return (
      <Light lightserver={props.lightserver} key={light.id} light={light} />
    );
  });

  return <div>{lightsItem}</div>;
};

class App extends React.Component {
  constructor(props) {
    super(props);

    this.state = {
      lights: [],
      lightserver: this.props.lightserver
    };

    const config = {
      method: "get",
      url: `${this.state.lightserver}/light`,
      headers: {
        //'Access-Control-Allow-Origin': '*',
      }
    };

    axios.request(config).then(res => {
      this.setState({ lights: res.data });
    });
  }

  render() {
    return (
      <div>
        <div>Lights: {this.state.lights.length}</div>
        <LightsArray
          lightserver={this.state.lightserver}
          lights={this.state.lights}
        />
      </div>
    );
  }
}

ReactDOM.render(
  <App lightserver="" />,
  document.querySelector(".container")
);
