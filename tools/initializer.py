import os
from google.cloud import pubsub_v1
from google.api_core.exceptions import AlreadyExists, NotFound

from conf import PROJECT_ID


# Setup the environment variables
os.environ["PUBSUB_EMULATOR_HOST"] = "localhost:8085"

# Define here the structure: { "topic_name": ["subscription_name1", "subscription_name2"], ... }
SCHEMA = {
    "api-req": ["sub-api-req"],
    "api-res": ["sub-api-res"],
}


def create_pubsub_resources():
    publisher = pubsub_v1.PublisherClient()
    subscriber = pubsub_v1.SubscriberClient()

    print(f"🚀 Starting Pub/Sub configuration for project: {PROJECT_ID}\n")

    for topic_id, subscriptions in SCHEMA.items():
        # 1. Topic creation
        topic_path = publisher.topic_path(PROJECT_ID, topic_id)
        
        try:
            publisher.create_topic(request={"name": topic_path})
            print(f"✅ Topic created: {topic_id}")
        except AlreadyExists:
            print(f"ℹ️  Topic exists: {topic_id}")
        except Exception as e:
            print(f"❌ Error creating topic {topic_id}: {e}")

        # 2. Subscription creation for this Topic
        for sub_id in subscriptions:
            sub_path = subscriber.subscription_path(PROJECT_ID, sub_id)

            try:
                subscriber.create_subscription(
                    request={"name": sub_path, "topic": topic_path}
                )
                print(f"   └── ✅ Subscription created: {sub_id}")
            except AlreadyExists:
                print(f"   └── ℹ️  Subscription exists: {sub_id}")
            except NotFound:
                 print(f"   └── ❌ Error: Topic {topic_id} not found.")
            except Exception as e:
                print(f"   └── ❌ Error creating subscription {sub_id}: {e}")

    print("\n✨ Configuration completed.")


if __name__ == "__main__":
    create_pubsub_resources()