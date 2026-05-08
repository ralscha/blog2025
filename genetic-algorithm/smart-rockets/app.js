const WORLD_WIDTH = 1120;
const WORLD_HEIGHT = 640;
const GENOME_WIDTH = 420;
const GENOME_HEIGHT = 230;
const CHART_WIDTH = 1120;
const CHART_HEIGHT = 260;
const LIFESPAN = 220;
const TARGET = { x: WORLD_WIDTH / 2, y: 58, radius: 18 };
const START = { x: WORLD_WIDTH / 2, y: WORLD_HEIGHT - 42 };
const MAX_FORCE = 0.28;
const MAX_SPEED = 5.2;
const ROCKET_RADIUS = 5;
const ELITE_COUNT = 4;
const TOURNAMENT_SIZE = 5;
const HISTORY_LIMIT = 180;
const OBSTACLE_PHASE = 0.035;

const state = {
  running: false,
  generation: 0,
  tick: 0,
  population: [],
  evaluated: [],
  best: null,
  fastestArrival: null,
  history: [],
  animationId: 0,
};

const elements = {
  populationSize: document.getElementById("populationSize"),
  mutationRate: document.getElementById("mutationRate"),
  windStrength: document.getElementById("windStrength"),
  speed: document.getElementById("speed"),
  populationSizeValue: document.getElementById("populationSizeValue"),
  mutationRateValue: document.getElementById("mutationRateValue"),
  windStrengthValue: document.getElementById("windStrengthValue"),
  speedValue: document.getElementById("speedValue"),
  generationValue: document.getElementById("generationValue"),
  fitnessValue: document.getElementById("fitnessValue"),
  distanceValue: document.getElementById("distanceValue"),
  arrivalValue: document.getElementById("arrivalValue"),
  statusValue: document.getElementById("statusValue"),
  fuelValue: document.getElementById("fuelValue"),
  collisionValue: document.getElementById("collisionValue"),
  fastestValue: document.getElementById("fastestValue"),
  diversityValue: document.getElementById("diversityValue"),
  startButton: document.getElementById("startButton"),
  pauseButton: document.getElementById("pauseButton"),
  resetButton: document.getElementById("resetButton"),
  worldCanvas: document.getElementById("worldCanvas"),
  genomeCanvas: document.getElementById("genomeCanvas"),
  historyCanvas: document.getElementById("historyCanvas"),
  pathsCanvas: document.getElementById("pathsCanvas"),
};

const worldCtx = elements.worldCanvas.getContext("2d");
const genomeCtx = elements.genomeCanvas.getContext("2d");
const historyCtx = elements.historyCanvas.getContext("2d");
const pathsCtx = elements.pathsCanvas.getContext("2d");

function populationSize() {
  return Number(elements.populationSize.value);
}

function mutationRate() {
  return Number(elements.mutationRate.value) / 100;
}

function windStrength() {
  return Number(elements.windStrength.value) / 1000;
}

function speed() {
  return Number(elements.speed.value);
}

function rand(min, max) {
  return min + Math.random() * (max - min);
}

function randInt(min, maxExclusive) {
  return Math.floor(rand(min, maxExclusive));
}

function randomForce() {
  const angle = rand(0, Math.PI * 2);
  const magnitude = rand(MAX_FORCE * 0.12, MAX_FORCE);
  return {
    x: Math.cos(angle) * magnitude,
    y: Math.sin(angle) * magnitude,
  };
}

function createGenome() {
  return Array.from({ length: LIFESPAN }, randomForce);
}

function cloneGenome(genome) {
  return genome.map((force) => ({ x: force.x, y: force.y }));
}

function createCandidate(genome = createGenome()) {
  return {
    genome,
    fitness: 0,
    minDistance: Number.POSITIVE_INFINITY,
    arrivalTick: null,
    fuelUsed: 0,
    collisions: 0,
    path: [],
    alive: true,
    reached: false,
  };
}

function updateOutputLabels() {
  elements.populationSizeValue.textContent = String(populationSize());
  elements.mutationRateValue.textContent = `${Math.round(mutationRate() * 100)}%`;
  elements.windStrengthValue.textContent = String(
    Number(elements.windStrength.value),
  );
  elements.speedValue.textContent = `${speed()}x`;
}

function movingObstacles(tick) {
  const gapWidth = 170;
  const topShift = Math.sin(tick * OBSTACLE_PHASE) * 150;
  const midShift = Math.sin(tick * OBSTACLE_PHASE + 1.7) * 210;
  const lowerShift = Math.sin(tick * OBSTACLE_PHASE + 3.1) * 135;
  return [
    { y: 170, height: 28, gapCenter: WORLD_WIDTH * 0.5 + topShift, gapWidth },
    {
      y: 315,
      height: 30,
      gapCenter: WORLD_WIDTH * 0.5 + midShift,
      gapWidth: 145,
    },
    {
      y: 460,
      height: 28,
      gapCenter: WORLD_WIDTH * 0.5 + lowerShift,
      gapWidth: 190,
    },
  ];
}

function windAt(position, tick) {
  const strength = windStrength();
  const wave =
    Math.sin(position.y * 0.018 + tick * 0.05) +
    Math.cos(position.x * 0.012 - tick * 0.035);
  return {
    x: wave * strength,
    y: Math.sin(position.x * 0.01 + tick * 0.025) * strength * 0.35,
  };
}

function hitsObstacle(position, tick) {
  return movingObstacles(tick).some((obstacle) => {
    const inBand =
      position.y + ROCKET_RADIUS > obstacle.y &&
      position.y - ROCKET_RADIUS < obstacle.y + obstacle.height;
    if (!inBand) {
      return false;
    }
    const gapLeft = obstacle.gapCenter - obstacle.gapWidth / 2;
    const gapRight = obstacle.gapCenter + obstacle.gapWidth / 2;
    return (
      position.x - ROCKET_RADIUS < gapLeft ||
      position.x + ROCKET_RADIUS > gapRight
    );
  });
}

function outOfBounds(position) {
  return (
    position.x < ROCKET_RADIUS ||
    position.x > WORLD_WIDTH - ROCKET_RADIUS ||
    position.y < ROCKET_RADIUS ||
    position.y > WORLD_HEIGHT - ROCKET_RADIUS
  );
}

function evaluateCandidate(candidate) {
  let position = { x: START.x, y: START.y };
  let velocity = { x: 0, y: 0 };
  let alive = true;
  let reached = false;
  let minDistance = distance(position, TARGET);
  let arrivalTick = null;
  let collisions = 0;
  let fuelUsed = 0;
  const path = [];

  for (let tick = 0; tick < LIFESPAN; tick++) {
    const force = candidate.genome[tick];
    fuelUsed += Math.hypot(force.x, force.y) / MAX_FORCE;

    if (alive && !reached) {
      const wind = windAt(position, tick);
      velocity.x += force.x + wind.x;
      velocity.y += force.y + wind.y;

      const speedValue = Math.hypot(velocity.x, velocity.y);
      if (speedValue > MAX_SPEED) {
        velocity.x = (velocity.x / speedValue) * MAX_SPEED;
        velocity.y = (velocity.y / speedValue) * MAX_SPEED;
      }

      position.x += velocity.x;
      position.y += velocity.y;

      if (hitsObstacle(position, tick) || outOfBounds(position)) {
        collisions += 1;
        alive = false;
      }

      const currentDistance = distance(position, TARGET);
      minDistance = Math.min(minDistance, currentDistance);
      if (currentDistance <= TARGET.radius) {
        reached = true;
        arrivalTick = tick;
      }
    }

    path.push({
      x: position.x,
      y: position.y,
      vx: velocity.x,
      vy: velocity.y,
      alive,
      reached,
    });
  }

  const inverseDistance = 900000 / Math.pow(minDistance + 10, 2);
  const arrivalBonus =
    arrivalTick === null ? 0 : 900 + (LIFESPAN - arrivalTick) * 6;
  const fuelBonus = Math.max(0, 160 - fuelUsed * 0.65);
  const collisionPenalty = collisions * 160;
  const progressBonus =
    Math.max(0, START.y - path[Math.min(path.length - 1, LIFESPAN - 1)].y) *
    0.28;
  const fitness = Math.max(
    0.01,
    inverseDistance +
      arrivalBonus +
      fuelBonus +
      progressBonus -
      collisionPenalty,
  );

  return {
    ...candidate,
    fitness,
    minDistance,
    arrivalTick,
    fuelUsed,
    collisions,
    path,
    alive,
    reached,
  };
}

function evaluatePopulation(population) {
  return population
    .map(evaluateCandidate)
    .sort((left, right) => right.fitness - left.fitness);
}

function nextGeneration(evaluated) {
  const next = [];
  const elites = evaluated.slice(0, ELITE_COUNT);
  for (const elite of elites) {
    next.push(createCandidate(cloneGenome(elite.genome)));
  }

  while (next.length < populationSize()) {
    const parentA = tournamentSelect(evaluated);
    const parentB = tournamentSelect(evaluated);
    const childGenome = mutate(crossover(parentA.genome, parentB.genome));
    next.push(createCandidate(childGenome));
  }

  return next;
}

function tournamentSelect(population) {
  let best = population[randInt(0, population.length)];
  for (let draw = 1; draw < TOURNAMENT_SIZE; draw++) {
    const challenger = population[randInt(0, population.length)];
    if (challenger.fitness > best.fitness) {
      best = challenger;
    }
  }
  return best;
}

function crossover(parentA, parentB) {
  const cut = randInt(12, LIFESPAN - 12);
  const child = [];
  for (let index = 0; index < LIFESPAN; index++) {
    const source = index < cut ? parentA : parentB;
    child.push({ x: source[index].x, y: source[index].y });
  }
  return child;
}

function mutate(genome) {
  const rate = mutationRate();
  return genome.map((force, index) => {
    if (Math.random() >= rate) {
      return force;
    }

    if (Math.random() < 0.22 && index > 0) {
      return {
        x: clamp(
          force.x * 0.55 + genome[index - 1].x * 0.45 + rand(-0.04, 0.04),
          -MAX_FORCE,
          MAX_FORCE,
        ),
        y: clamp(
          force.y * 0.55 + genome[index - 1].y * 0.45 + rand(-0.04, 0.04),
          -MAX_FORCE,
          MAX_FORCE,
        ),
      };
    }

    const replacement = randomForce();
    return {
      x: clamp(force.x * 0.45 + replacement.x * 0.55, -MAX_FORCE, MAX_FORCE),
      y: clamp(force.y * 0.45 + replacement.y * 0.55, -MAX_FORCE, MAX_FORCE),
    };
  });
}

function resetSimulation() {
  state.running = false;
  state.generation = 0;
  state.tick = 0;
  state.population = Array.from({ length: populationSize() }, () =>
    createCandidate(),
  );
  state.evaluated = evaluatePopulation(state.population);
  state.best = state.evaluated[0];
  state.fastestArrival = state.best.arrivalTick;
  state.history = [];
  recordHistory();
  renderAll();
}

function evolveOneGeneration() {
  state.population = nextGeneration(state.evaluated);
  state.evaluated = evaluatePopulation(state.population);
  state.best = state.evaluated[0];
  state.generation += 1;
  state.tick = 0;

  if (state.best.arrivalTick !== null) {
    if (
      state.fastestArrival === null ||
      state.best.arrivalTick < state.fastestArrival
    ) {
      state.fastestArrival = state.best.arrivalTick;
    }
  }

  recordHistory();
}

function recordHistory() {
  const averageFitness = average(
    state.evaluated,
    (candidate) => candidate.fitness,
  );
  state.history.push({
    generation: state.generation,
    best: state.best.fitness,
    average: averageFitness,
    distance: state.best.minDistance,
  });
  if (state.history.length > HISTORY_LIMIT) {
    state.history.shift();
  }
}

function animate() {
  if (state.running) {
    for (let step = 0; step < speed(); step++) {
      state.tick += 1;
      if (state.tick >= LIFESPAN) {
        evolveOneGeneration();
      }
    }
  }

  renderAll();
  state.animationId = requestAnimationFrame(animate);
}

function renderAll() {
  updateStats();
  drawWorld();
  drawGenome();
  drawHistory();
  drawPaths();
}

function updateStats() {
  const best = state.best;
  const arrival =
    best.arrivalTick === null ? "None" : `Frame ${best.arrivalTick}`;
  const fastest =
    state.fastestArrival === null ? "None" : `Frame ${state.fastestArrival}`;
  elements.generationValue.textContent = String(state.generation);
  elements.fitnessValue.textContent = best.fitness.toFixed(1);
  elements.distanceValue.textContent = `${best.minDistance.toFixed(0)} px`;
  elements.arrivalValue.textContent = arrival;
  elements.fuelValue.textContent = `${Math.min(999, (best.fuelUsed / LIFESPAN) * 100).toFixed(0)}%`;
  elements.collisionValue.textContent = String(best.collisions);
  elements.fastestValue.textContent = fastest;
  elements.diversityValue.textContent = averageGenomeDistance(
    state.evaluated,
  ).toFixed(2);
  elements.statusValue.textContent = statusText(best);
}

function statusText(best) {
  if (best.reached) {
    return "A viable path reached the target; evolution is now polishing speed and fuel.";
  }
  if (best.minDistance < 70) {
    return "The population is threading the final gap but has not stabilized arrival yet.";
  }
  if (state.generation < 3) {
    return "Fresh genomes are mostly random thrust, collisions, and missed gates.";
  }
  return "Selection is preserving arcs that survive the wind and moving walls.";
}

function drawWorld() {
  const ctx = worldCtx;
  ctx.clearRect(0, 0, WORLD_WIDTH, WORLD_HEIGHT);
  drawBackground(ctx, WORLD_WIDTH, WORLD_HEIGHT);
  drawWindField(ctx, state.tick);
  drawObstacles(ctx, state.tick);
  drawTarget(ctx);
  drawPopulationTrails(ctx);
  drawRocketReplay(ctx, state.best, state.tick);
  drawStart(ctx);
}

function drawBackground(ctx, width, height) {
  const gradient = ctx.createLinearGradient(0, 0, 0, height);
  gradient.addColorStop(0, "#101922");
  gradient.addColorStop(0.52, "#101419");
  gradient.addColorStop(1, "#171419");
  ctx.fillStyle = gradient;
  ctx.fillRect(0, 0, width, height);

  ctx.strokeStyle = "rgba(255, 255, 255, 0.045)";
  ctx.lineWidth = 1;
  for (let x = 40; x < width; x += 80) {
    ctx.beginPath();
    ctx.moveTo(x, 0);
    ctx.lineTo(x, height);
    ctx.stroke();
  }
  for (let y = 40; y < height; y += 80) {
    ctx.beginPath();
    ctx.moveTo(0, y);
    ctx.lineTo(width, y);
    ctx.stroke();
  }
}

function drawWindField(ctx, tick) {
  ctx.save();
  ctx.strokeStyle = "rgba(92, 183, 255, 0.28)";
  ctx.lineWidth = 1.5;
  for (let y = 92; y < WORLD_HEIGHT - 70; y += 74) {
    for (let x = 70; x < WORLD_WIDTH - 40; x += 104) {
      const wind = windAt({ x, y }, tick);
      const scale = 4200;
      const endX = x + wind.x * scale;
      const endY = y + wind.y * scale;
      ctx.beginPath();
      ctx.moveTo(x, y);
      ctx.lineTo(endX, endY);
      ctx.stroke();
      ctx.beginPath();
      ctx.arc(endX, endY, 2, 0, Math.PI * 2);
      ctx.fillStyle = "rgba(92, 183, 255, 0.45)";
      ctx.fill();
    }
  }
  ctx.restore();
}

function drawObstacles(ctx, tick) {
  for (const obstacle of movingObstacles(tick)) {
    const gapLeft = obstacle.gapCenter - obstacle.gapWidth / 2;
    const gapRight = obstacle.gapCenter + obstacle.gapWidth / 2;
    const gradient = ctx.createLinearGradient(
      0,
      obstacle.y,
      0,
      obstacle.y + obstacle.height,
    );
    gradient.addColorStop(0, "rgba(251, 113, 133, 0.86)");
    gradient.addColorStop(1, "rgba(153, 55, 72, 0.86)");
    ctx.fillStyle = gradient;
    roundRect(ctx, 0, obstacle.y, Math.max(0, gapLeft), obstacle.height, 5);
    ctx.fill();
    roundRect(
      ctx,
      gapRight,
      obstacle.y,
      WORLD_WIDTH - gapRight,
      obstacle.height,
      5,
    );
    ctx.fill();

    ctx.fillStyle = "rgba(104, 211, 145, 0.18)";
    roundRect(
      ctx,
      gapLeft,
      obstacle.y + 4,
      obstacle.gapWidth,
      obstacle.height - 8,
      4,
    );
    ctx.fill();
  }
}

function drawTarget(ctx) {
  ctx.save();
  ctx.translate(TARGET.x, TARGET.y);
  ctx.strokeStyle = "rgba(104, 211, 145, 0.88)";
  ctx.lineWidth = 3;
  ctx.beginPath();
  ctx.arc(0, 0, TARGET.radius, 0, Math.PI * 2);
  ctx.stroke();
  ctx.beginPath();
  ctx.arc(
    0,
    0,
    TARGET.radius + 12 + Math.sin(Date.now() * 0.006) * 3,
    0,
    Math.PI * 2,
  );
  ctx.strokeStyle = "rgba(104, 211, 145, 0.22)";
  ctx.stroke();
  ctx.fillStyle = "#68d391";
  ctx.beginPath();
  ctx.arc(0, 0, 4, 0, Math.PI * 2);
  ctx.fill();
  ctx.restore();
}

function drawStart(ctx) {
  ctx.save();
  ctx.fillStyle = "rgba(242, 184, 75, 0.22)";
  roundRect(ctx, START.x - 48, START.y + 16, 96, 10, 5);
  ctx.fill();
  ctx.fillStyle = "#f2b84b";
  ctx.beginPath();
  ctx.arc(START.x, START.y, 5, 0, Math.PI * 2);
  ctx.fill();
  ctx.restore();
}

function drawPopulationTrails(ctx) {
  const sample = state.evaluated.slice(0, Math.min(28, state.evaluated.length));
  sample.reverse().forEach((candidate, index) => {
    drawPath(
      ctx,
      candidate.path,
      `rgba(92, 183, 255, ${0.05 + index * 0.006})`,
      1.2,
      state.tick,
    );
  });
}

function drawRocketReplay(ctx, candidate, tick) {
  if (!candidate || candidate.path.length === 0) {
    return;
  }
  drawPath(ctx, candidate.path, "rgba(242, 184, 75, 0.8)", 3, tick);
  const frame = candidate.path[Math.min(tick, candidate.path.length - 1)];
  drawRocket(ctx, frame, "#f2b84b");
}

function drawPath(ctx, path, color, width, until = path.length - 1) {
  if (!path || path.length < 2) {
    return;
  }
  const last = Math.min(until, path.length - 1);
  ctx.save();
  ctx.strokeStyle = color;
  ctx.lineWidth = width;
  ctx.beginPath();
  ctx.moveTo(path[0].x, path[0].y);
  for (let index = 1; index <= last; index++) {
    ctx.lineTo(path[index].x, path[index].y);
  }
  ctx.stroke();
  ctx.restore();
}

function drawRocket(ctx, frame, color) {
  const angle = Math.atan2(frame.vy, frame.vx) + Math.PI / 2;
  ctx.save();
  ctx.translate(frame.x, frame.y);
  ctx.rotate(angle);
  ctx.fillStyle = color;
  ctx.strokeStyle = "rgba(0, 0, 0, 0.35)";
  ctx.lineWidth = 1.5;
  ctx.beginPath();
  ctx.moveTo(0, -12);
  ctx.lineTo(7, 9);
  ctx.lineTo(0, 5);
  ctx.lineTo(-7, 9);
  ctx.closePath();
  ctx.fill();
  ctx.stroke();
  if (!frame.reached && frame.alive) {
    ctx.fillStyle = "rgba(251, 113, 133, 0.72)";
    ctx.beginPath();
    ctx.moveTo(-3, 9);
    ctx.lineTo(0, 18 + Math.sin(Date.now() * 0.03) * 4);
    ctx.lineTo(3, 9);
    ctx.closePath();
    ctx.fill();
  }
  ctx.restore();
}

function drawGenome() {
  const ctx = genomeCtx;
  const width = GENOME_WIDTH;
  const height = GENOME_HEIGHT;
  ctx.clearRect(0, 0, width, height);
  drawBackground(ctx, width, height);

  const genome = state.best.genome;
  const barWidth = width / genome.length;
  for (let index = 0; index < genome.length; index++) {
    const force = genome[index];
    const magnitude = Math.hypot(force.x, force.y) / MAX_FORCE;
    const angle = Math.atan2(force.y, force.x);
    const hue = 190 + Math.sin(angle) * 45;
    ctx.fillStyle = `hsla(${hue}, 78%, ${42 + magnitude * 28}%, 0.82)`;
    const barHeight = Math.max(2, magnitude * (height - 36));
    ctx.fillRect(
      index * barWidth,
      height - barHeight,
      Math.max(1, barWidth + 0.5),
      barHeight,
    );
  }

  ctx.strokeStyle = "rgba(242, 184, 75, 0.85)";
  ctx.lineWidth = 2;
  ctx.beginPath();
  for (let index = 0; index < genome.length; index += 3) {
    const x = (index / (genome.length - 1)) * width;
    const y = height / 2 + (genome[index].y / MAX_FORCE) * (height * 0.38);
    if (index === 0) {
      ctx.moveTo(x, y);
    } else {
      ctx.lineTo(x, y);
    }
  }
  ctx.stroke();
}

function drawHistory() {
  const ctx = historyCtx;
  const width = CHART_WIDTH;
  const height = CHART_HEIGHT;
  ctx.clearRect(0, 0, width, height);
  drawBackground(ctx, width, height);

  if (state.history.length < 2) {
    return;
  }

  const maxFitness = Math.max(...state.history.map((point) => point.best), 1);
  drawSeries(
    ctx,
    state.history,
    (point) => point.best,
    maxFitness,
    "#68d391",
    width,
    height,
  );
  drawSeries(
    ctx,
    state.history,
    (point) => point.average,
    maxFitness,
    "#5cb7ff",
    width,
    height,
  );

  ctx.fillStyle = "rgba(243, 239, 231, 0.68)";
  ctx.font = "700 13px Inter, sans-serif";
  ctx.fillText("best", 18, 24);
  ctx.fillStyle = "#68d391";
  ctx.fillRect(58, 15, 18, 4);
  ctx.fillStyle = "rgba(243, 239, 231, 0.68)";
  ctx.fillText("average", 92, 24);
  ctx.fillStyle = "#5cb7ff";
  ctx.fillRect(158, 15, 18, 4);
}

function drawSeries(ctx, data, selector, maxValue, color, width, height) {
  ctx.save();
  ctx.strokeStyle = color;
  ctx.lineWidth = 2.5;
  ctx.beginPath();
  data.forEach((point, index) => {
    const x = (index / (data.length - 1)) * (width - 36) + 18;
    const y = height - 24 - (selector(point) / maxValue) * (height - 54);
    if (index === 0) {
      ctx.moveTo(x, y);
    } else {
      ctx.lineTo(x, y);
    }
  });
  ctx.stroke();
  ctx.restore();
}

function drawPaths() {
  const ctx = pathsCtx;
  const width = CHART_WIDTH;
  const height = CHART_HEIGHT;
  ctx.clearRect(0, 0, width, height);
  drawBackground(ctx, width, height);

  const scaleX = width / WORLD_WIDTH;
  const scaleY = height / WORLD_HEIGHT;
  ctx.save();
  ctx.scale(scaleX, scaleY);
  state.evaluated
    .slice(0, 40)
    .reverse()
    .forEach((candidate, index) => {
      const alpha = 0.06 + index * 0.006;
      const color = candidate.reached
        ? `rgba(104, 211, 145, ${alpha + 0.08})`
        : `rgba(92, 183, 255, ${alpha})`;
      drawPath(ctx, candidate.path, color, 1.6, candidate.path.length - 1);
    });
  drawTarget(ctx);
  drawStart(ctx);
  ctx.restore();
}

function averageGenomeDistance(population) {
  if (population.length < 2) {
    return 0;
  }
  const best = population[0].genome;
  const sample = population.slice(1, Math.min(population.length, 24));
  return average(sample, (candidate) => {
    let total = 0;
    for (let index = 0; index < LIFESPAN; index += 8) {
      total += Math.hypot(
        best[index].x - candidate.genome[index].x,
        best[index].y - candidate.genome[index].y,
      );
    }
    return total;
  });
}

function average(items, selector) {
  if (items.length === 0) {
    return 0;
  }
  return items.reduce((sum, item) => sum + selector(item), 0) / items.length;
}

function distance(left, right) {
  return Math.hypot(left.x - right.x, left.y - right.y);
}

function clamp(value, min, max) {
  return Math.max(min, Math.min(max, value));
}

function roundRect(ctx, x, y, width, height, radius) {
  const safeRadius = Math.min(
    radius,
    Math.abs(width) / 2,
    Math.abs(height) / 2,
  );
  ctx.beginPath();
  ctx.moveTo(x + safeRadius, y);
  ctx.lineTo(x + width - safeRadius, y);
  ctx.quadraticCurveTo(x + width, y, x + width, y + safeRadius);
  ctx.lineTo(x + width, y + height - safeRadius);
  ctx.quadraticCurveTo(
    x + width,
    y + height,
    x + width - safeRadius,
    y + height,
  );
  ctx.lineTo(x + safeRadius, y + height);
  ctx.quadraticCurveTo(x, y + height, x, y + height - safeRadius);
  ctx.lineTo(x, y + safeRadius);
  ctx.quadraticCurveTo(x, y, x + safeRadius, y);
}

function resizeCanvases() {
  for (const canvas of [
    elements.worldCanvas,
    elements.genomeCanvas,
    elements.historyCanvas,
    elements.pathsCanvas,
  ]) {
    const displayWidth = canvas.clientWidth;
    const displayHeight = canvas.clientHeight;
    const ratio = window.devicePixelRatio || 1;
    canvas.width = Math.max(1, Math.round(displayWidth * ratio));
    canvas.height = Math.max(1, Math.round(displayHeight * ratio));
    const ctx = canvas.getContext("2d");
    const baseWidth =
      canvas === elements.worldCanvas
        ? WORLD_WIDTH
        : canvas === elements.genomeCanvas
          ? GENOME_WIDTH
          : CHART_WIDTH;
    const baseHeight =
      canvas === elements.worldCanvas
        ? WORLD_HEIGHT
        : canvas === elements.genomeCanvas
          ? GENOME_HEIGHT
          : CHART_HEIGHT;
    ctx.setTransform(
      canvas.width / baseWidth,
      0,
      0,
      canvas.height / baseHeight,
      0,
      0,
    );
  }
  renderAll();
}

function attachEvents() {
  elements.startButton.addEventListener("click", () => {
    state.running = true;
  });
  elements.pauseButton.addEventListener("click", () => {
    state.running = false;
  });
  elements.resetButton.addEventListener("click", resetSimulation);

  for (const input of [
    elements.populationSize,
    elements.mutationRate,
    elements.windStrength,
    elements.speed,
  ]) {
    input.addEventListener("input", () => {
      updateOutputLabels();
      if (input === elements.populationSize) {
        resetSimulation();
      }
    });
  }

  window.addEventListener("resize", resizeCanvases);
}

updateOutputLabels();
attachEvents();
resetSimulation();
resizeCanvases();
animate();
