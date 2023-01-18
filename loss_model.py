# two stage loss markov model to model bursty and random losses in a network

import numpy as np

# define the loss model
class LossModel:
    def __init__(self, bursty_loss_rate, random_loss_rate, bursty_loss_duration, random_loss_duration):
        self.bursty_loss_rate = bursty_loss_rate
        self.random_loss_rate = random_loss_rate
        self.bursty_loss_duration = bursty_loss_duration
        self.random_loss_duration = random_loss_duration

    def get_loss(self, time):
        bursty_loss = np.random.binomial(1, self.bursty_loss_rate, time)
        random_loss = np.random.binomial(1, self.random_loss_rate, time)
        bursty_loss_duration = np.random.exponential(self.bursty_loss_duration, time)
        random_loss_duration = np.random.exponential(self.random_loss_duration, time)

        bursty_loss = np.multiply(bursty_loss, bursty_loss_duration)
        random_loss = np.multiply(random_loss, random_loss_duration)

        loss = np.add(bursty_loss, random_loss)
        loss = np.where(loss > 1, 1, loss)

        return loss

# define the loss model
bursty_loss_rate = 0.1
random_loss_rate = 0.1
bursty_loss_duration = 0.1
random_loss_duration = 0.1

loss_model = LossModel(bursty_loss_rate, random_loss_rate, bursty_loss_duration, random_loss_duration)

# get the loss
time = 1000
loss = loss_model.get_loss(time)

