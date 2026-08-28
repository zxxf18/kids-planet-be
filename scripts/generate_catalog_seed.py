#!/usr/bin/env python3
"""Generate DB-only Chinese titles and content tags from the local 426-song corpus."""

from __future__ import annotations

import argparse
import re
from pathlib import Path


BASE_ZH = {
    "Five Little Monkeys": "五只小猴子",
    "Twinkle Twinkle Little Star": "一闪一闪小星星",
    "Hickory Dickory...Crash!": "滴答滴答，哐当！",
    "Row Row Row Your Boat": "划呀划小船",
    "BINGO": "宾果小狗",
    "S-A-N-T-A": "圣诞老人拼字歌",
    "The Months Chant": "月份歌谣",
    "Ten In The Bed": "床上有十个",
    "Little Snowflake": "小雪花",
    "Open Shut Them": "张开再合上",
    "Jingle Bells": "铃儿响叮当",
    "Head Shoulders Knees - Toes": "头、肩膀、膝盖和脚趾",
    "Head Shoulders Knees And Toes": "头、肩膀、膝盖和脚趾",
    "Sweet Dreams": "甜甜的梦",
    "Old MacDonald": "老麦克唐纳",
    "Old MacDonald Had A Farm": "老麦克唐纳有个农场",
    "Walking In The Jungle": "漫步丛林",
    "Walking In The Forest": "漫步森林",
    "The Animals On The Farm": "农场里的动物",
    "Do You Like Broccoli Ice Cream?": "你喜欢西兰花冰淇淋吗？",
    "Count - Move": "边数边动",
    "Count & Move": "边数边动",
    "If You're Happy": "如果你开心",
    "How's The Weather?": "天气怎么样？",
    "How's The Weather": "天气怎么样",
    "One Little Finger": "一根小手指",
    "Counting Bananas": "数香蕉",
    "One Potato, Two Potatoes": "一个土豆，两个土豆",
    "I See Something Blue": "我看见蓝色",
    "I See Something Pink": "我看见粉色",
    "Let's Go To The Zoo": "我们去动物园",
    "The Wheels On The Bus": "巴士上的轮子",
    "Mary Had A Kangaroo": "玛丽有只小袋鼠",
    "I Have A Pet": "我有一只宠物",
    "Go Away!": "快走开！",
    "Can You Make A Happy Face?": "你能做个开心的表情吗？",
    "Five Creepy Spiders": "五只吓人的蜘蛛",
    "Knock Knock, Trick Or Treat?": "咚咚敲门，不给糖就捣蛋",
    "Knock Knock, Trick Or Treat": "咚咚敲门，不给糖就捣蛋",
    "This Is The Way We Carve A Pumpkin": "我们这样雕南瓜",
    "Who Took The Candy?": "谁拿走了糖果？",
    "One For You, One For Me": "一个给你，一个给我",
    "Go Away, Spooky Goblin!": "吓人小妖怪，快走开！",
    "Put On Your Shoes": "穿上你的鞋",
    "Put On Your Boots": "穿上你的靴子",
    "We Wish You A Merry Christmas": "祝你圣诞快乐",
    "Decorate The Christmas Tree": "装饰圣诞树",
    "What Do You Want For Christmas?": "圣诞节你想要什么？",
    "Santa's On His Way": "圣诞老人来啦",
    "Count Down - Move": "倒数着动起来",
    "I'm A Little Snowman": "我是小雪人",
    "Skidamarink": "我爱你呀",
    "Skidamarink A Dink A Dink": "我爱你呀",
    "My Teddy Bear": "我的泰迪熊",
    "Yes, I Can!": "是的，我可以！",
    "Wag Your Tail": "摇摇尾巴",
    "What Do You Hear?": "你听见了什么？",
    "The Itsy Bitsy Spider": "小小蜘蛛",
    "Rain Rain Go Away": "雨呀雨，快走开",
    "Rock Scissors Paper": "石头剪刀布",
    "The Shape Song": "形状歌",
    "Do You Like Spaghetti Yogurt?": "你喜欢意面酸奶吗？",
    "The Bath Song": "洗澡歌",
    "Who Took The Cookie?": "谁拿走了饼干？",
    "Say Cheese!": "笑一个！",
    "The Pinocchio": "匹诺曹",
    "We All Fall Down": "我们一起倒下来",
    "Make A Circle": "围成一个圈",
    "Give Me Something Good To Eat": "给我一点好吃的",
    "Five Little Pumpkins": "五个小南瓜",
    "Goodbye, My Friends": "再见，我的朋友",
    "Hello, Trick Or Treat?": "你好，不给糖就捣蛋",
    "Hello Reindeer, Goodbye Snowman": "你好小驯鹿，再见小雪人",
    "Goodbye, Snowman": "再见，小雪人",
    "Hello, Reindeer": "你好，小驯鹿",
    "Hello, My Friends": "你好，我的朋友",
    "See You Later, Alligator": "回头见，小鳄鱼",
    "After A While, Crocodile": "一会儿见，大鳄鱼",
    "Bye Bye Goodbye": "拜拜，再见",
    "Good Morning, Mr. Rooster": "早上好，公鸡先生",
    "Hello Hello!": "你好，你好！",
    "Hello!": "你好！",
    "Clean Up!": "收拾好！",
    "Uh-huh!": "嗯哼！",
    "10 Little Elves": "十个小精灵",
    "Jingle Jingle Little Bell": "小铃铛叮叮当",
    "Santa, Where Are You?": "圣诞老人，你在哪里？",
    "Mystery Box": "神秘盒子",
    "Do You Like Pickle Pudding?": "你喜欢酸黄瓜布丁吗？",
    "Seven Steps": "七步歌",
    "Walking Walking": "走呀走",
    "Do You Like Lasagna Milkshakes?": "你喜欢千层面奶昔吗？",
    "Days Of The Week": "星期歌",
    "This Is The Way": "我们就是这样做",
    "Hide And Seek": "捉迷藏",
    "The Alphabet Chant": "字母歌谣",
    "The Alphabet Song": "字母歌",
    "The Alphabet Swing": "字母摇摆歌",
    "The Alphabet Rhyme": "字母童谣",
    "The Alphabet Is So Much Fun": "字母真有趣",
    "Five Little Ducks": "五只小鸭子",
    "Six Little Ducks": "六只小鸭子",
    "The Skeleton Dance": "骷髅舞",
    "How Many Fingers?": "有几根手指？",
    "Five Little Speckled Frogs": "五只斑点小青蛙",
    "The Ice Cream Song": "冰淇淋歌",
    "Baby Shark": "鲨鱼宝宝",
    "Mr. Golden Sun": "金太阳先生",
    "Apples - Bananas": "苹果和香蕉",
    "10 Little Dinosaurs": "十只小恐龙",
    "Little Robin Redbreast": "红胸小知更鸟",
    "Peekaboo": "躲猫猫",
    "Alice The Camel": "骆驼爱丽丝",
    "Brush Your Teeth": "刷刷牙",
    "The Muffin Man": "松饼师傅",
    "The Ants Go Marching": "蚂蚁行军",
    "10 Little Airplanes": "十架小飞机",
    "I Like You": "我喜欢你",
    "Down By The Bay": "海湾边",
    "Down In The Bay": "海湾里",
    "Jack - Jill": "杰克和吉尔",
    "Jack & Jill": "杰克和吉尔",
    "10 Little Sailboats": "十艘小帆船",
    "Humpty Dumpty": "蛋头先生",
    "What Do You Like To Do?": "你喜欢做什么？",
    "Peanut Butter - Jelly": "花生酱和果冻",
    "I Can't Remember The Words To This Song": "我记不住这首歌的歌词",
    "Five Little Monsters Jumping On The Bed": "五只小怪兽在床上跳",
    "Baby Shark Halloween": "鲨鱼宝宝过万圣节",
    "Take Me Out To The Ball Game": "带我去看球赛",
    "Pat-A-Cake": "拍拍蛋糕",
    "Wind The Bobbin Up": "绕线轴",
    "10 Little Tractors": "十辆小拖拉机",
    "Up On The Housetop": "在屋顶上",
    "12 Days Of Christmas": "圣诞节的十二天",
    "12 Days of Christmas": "圣诞节的十二天",
    "10 Little Fishies": "十条小鱼",
    "Follow Me": "跟我来",
    "The Farmer In The Dell": "山谷里的农夫",
    "10 Little Buses": "十辆小巴士",
    "This Is The Way We Get Dressed": "我们这样穿衣服",
    "Are You Hungry?": "你饿了吗？",
    "Are You Hungry": "你饿了吗",
    "Let's Take The Subway": "我们去坐地铁",
    "10 Little Bicycles": "十辆小自行车",
    "Hot Cross Buns": "热十字面包",
    "There's A Hole In The Bottom Of The Sea": "海底有个洞",
    "Here We Go Looby Loo": "一起来跳露比舞",
    "What's Your Favorite Color?": "你最喜欢什么颜色？",
    "This Is The Way We Go To Bed": "我们这样去睡觉",
    "Red Light, Green Light": "红灯停，绿灯行",
    "Red Yellow Green Blue": "红黄绿蓝",
    "Here Is The Beehive": "这里有个蜂巢",
    "When The Band Comes Marching In": "乐队进场时",
    "Over The Deep Blue Sea": "越过深蓝大海",
    "The Bees Go Buzzing": "蜜蜂嗡嗡飞",
    "Santa Shark": "圣诞鲨鱼",
    "What's This? What's That?": "这是什么？那是什么？",
    "If You're Happy And You Know It": "如果感到幸福你就拍拍手",
    "Six In The Bed": "床上有六个",
    "With My Heart": "用我的心",
    "Driving In My Car": "开着我的小汽车",
    "The Jellyfish": "小水母",
    "Peekaboo, I Love You": "躲猫猫，我爱你",
    "The More We Get Together": "我们越聚越开心",
    "Where Is Baby?": "宝宝在哪里？",
    "Sitting On The Potty": "坐在小马桶上",
    "What's Your Favorite Flavor Of Ice Cream?": "你最喜欢什么口味的冰淇淋？",
    "The Bear Went Over The Mountain": "小熊翻过山",
    "I Like To Ride My Bicycle": "我喜欢骑自行车",
    "Here Comes The Fire Truck": "消防车来啦",
    "Line Up!": "排好队！",
    "I Love The Mountains": "我爱群山",
    "Pop The Bubbles": "戳泡泡",
    "Halloween ABC Song": "万圣节字母歌",
    "My Happy Song": "我的快乐歌",
    "My Yellow Car": "我的黄色小汽车",
    "Are You Sleeping, Baby Bear?": "小熊宝宝，你睡了吗？",
    "At The North Pole": "在北极",
    "Silent Night": "平安夜",
    "And The Green Grass Grew": "绿草长呀长",
    "Butterfly Ladybug Bumblebee": "蝴蝶、瓢虫和大黄蜂",
    "Pizza Party": "披萨派对",
    "Let's Count To 100": "一起数到一百",
    "Star Light, Star Bright": "星光闪亮",
    "Once I Caught A Fish Alive": "我曾捉到一条活鱼",
    "Pink Purple Orange Brown": "粉紫橙棕",
    "Move!": "动起来！",
    "Me!": "这就是我！",
    "A Sailor Went To Sea": "水手出海去",
    "Five Little Ghosts": "五个小幽灵",
    "Down In The Deep Blue Sea": "在深蓝的大海里",
    "Three Little Kittens": "三只小猫咪",
    "What's Your Name?": "你叫什么名字？",
    "As Quiet As A Mouse": "像小老鼠一样安静",
    "This Is A Happy Face": "这是一张开心的脸",
    "The Rainbow Song": "彩虹歌",
    "Everything Is Going To Be Alright": "一切都会好起来",
    "When I Grow Up": "当我长大",
    "The Roly Poly Roll": "滚呀滚",
    "Hush Little Baby": "嘘，小宝宝",
    "There’s A Monster In My Tummy": "我的肚子里有只小怪兽",
    "There's A Monster In My Tummy": "我的肚子里有只小怪兽",
    "Milk And Cookies": "牛奶和饼干",
    "Teddy Bear, Teddy Bear": "泰迪熊，泰迪熊",
    "Beddy-Bye Butterfly": "晚安，小蝴蝶",
    "First We Wash Our Hands": "我们先洗手",
    "The Seasons Song": "四季歌",
    "Picked A Strawberry": "摘了一颗草莓",
    "Crawl Like A Caterpillar": "像毛毛虫一样爬",
    "We're Going On A Rocket Ship": "我们要坐火箭出发",
    "500 Ducks": "五百只鸭子",
    "The Fish Go Swimming": "鱼儿游呀游",
    "Happy Birthday To You": "祝你生日快乐",
    "This Is The Way We Make Friends": "我们这样交朋友",
    "ABC Quack": "字母嘎嘎歌",
    "ABC Boo": "字母幽灵歌",
    "ABC Hop": "字母跳跳歌",
    "Boom Chicka Boom": "砰恰砰节奏歌",
    "The Creepy Crawly Spider": "爬呀爬的小蜘蛛",
    "Monster Party": "怪兽派对",
    "I'm Thankful": "我心怀感恩",
    "Gingerbread House": "姜饼屋",
    "Nap Time": "午睡时间",
    "The Hand Washing Song": "洗手歌",
    "I Have A Friend": "我有一个朋友",
    "Raindrops Falling": "雨滴落下来",
    "Good Morning It’s Such A Beautiful Day": "早上好，多么美好的一天",
    "I Love The Ocean": "我爱大海",
    "Stand Up, Sit Down": "站起来，坐下去",
    "Shelly The Snail": "蜗牛雪莉",
    "Good Night Sleep Tight": "晚安，睡个好觉",
    "Can You Imagine": "你能想象吗",
    "Let's Go To The Beach": "我们去海滩",
    "Adding Up To 10": "加起来等于十",
    "Counting Up To 20": "数到二十",
    "I Spy": "我发现了",
    "I Am A Dinosaur": "我是一只恐龙",
    "Happy New Year": "新年快乐",
    "8 Little Planets": "八颗小行星",
    "May I Please, Yes You May": "我可以吗？当然可以",
    "Tucked In My Bed": "盖好被子睡觉",
    "I Love My Little Garbage Truck": "我爱我的小垃圾车",
    "My Glasses": "我的眼镜",
    "The Crabs Go Crawling": "螃蟹爬呀爬",
    "Little Birdie": "小鸟儿",
    "Let's Blow A Bubble": "我们来吹泡泡",
    "Sunny Day": "晴朗的一天",
    "Let's Take Turns": "我们轮流来",
    "Five Spotted Dogs": "五只斑点狗",
    "Little Animal Dance": "小动物舞会",
    "The Ducks Go Waddling": "鸭子摇摇摆摆",
    "It’s Way Too Hot Today": "今天实在太热啦",
    "One Foot Then The Next Foot": "一步接着一步",
    "Mixing Colors": "混合颜色",
    "Left And Right": "左边和右边",
    "Dance And Stop": "跳舞再停下",
    "Stomp": "跺跺脚",
    "A Is For Apple": "A 代表苹果",
    "You’re The Best At Being You": "做自己就是最棒的",
    "I Have A Loose Tooth": "我有一颗松动的牙",
    "Jingle Bells (Learn - Sing)": "学唱铃儿响叮当",
    "Head Shoulders Knees - Toes (Sing It)": "唱一唱头肩膝脚趾",
    "Head Shoulders Knees - Toes (Learn It)": "学一学头肩膝脚趾",
    "Head Shoulders Knees - Toes (Speeding Up)": "越来越快的头肩膝脚趾",
    "Who Took The Cookie? (On The Farm)": "农场里谁拿走了饼干？",
    "Peekaboo Playground": "游乐场躲猫猫",
    "Down By The Spooky Bay": "幽灵海湾边",
    "Peekaboo Halloween": "万圣节躲猫猫",
    "Do You Like Broccoli Ice Cream? (Puppets)": "你喜欢西兰花冰淇淋吗？（布偶版）",
    "Pass The Beanbag": "传沙包",
    "Where Is Thumbkin?": "大拇指在哪里？",
    "Peekaboo Christmas": "圣诞节躲猫猫",
    "If You're Happy And You Know It Shout Hoo-ray": "如果感到幸福就欢呼",
    "Baby Shark - Nursery Rhymes With Caitie": "凯蒂一起唱鲨鱼宝宝",
    "10 Monsters In The Bed": "床上有十只小怪兽",
    "This Is The Way We Trick Or Treat": "我们这样去讨糖",
    "10 Little Garbage Trucks": "十辆小垃圾车",
    "10 Little Fire Trucks": "十辆小消防车",
    "Peekaboo, Thank You": "躲猫猫，谢谢你",
    "Five Little Elves": "五个小精灵",
    "The Alphabet Swing (Lowercase Version)": "小写字母摇摆歌",
    "Who Took The Cookie? (Under The Sea)": "海底谁拿走了饼干？",
    "Toodly Doodly Doo": "嘟嘟啦啦歌",
    "Let's Go For A Walk Outside": "我们去户外散步",
    "10 Apples On My Head": "头顶十个苹果",
    "See You Later": "回头见",
    "Five Little Chicks": "五只小鸡",
    "Six Little Ghosts": "六个小幽灵",
    "If I Were A Ghost": "如果我是小幽灵",
    "Let's Decorate The House For Halloween": "一起装饰万圣节小屋",
    "Here You Are, Thank You": "给你，谢谢",
    "Let's Decorate Our Christmas Tree": "一起装饰圣诞树",
    "If You're Happy And You Know It Spin Around": "如果感到幸福就转个圈",
    "Let's Go For A Swim Outside": "我们去户外游泳",
    "ABC Song Speeding Up": "越来越快的字母歌",
    "The Great Christmas Tree Hunt": "寻找最棒的圣诞树",
    "Countdown To Christmas": "圣诞节倒计时",
    "Let's Make A Snowman": "一起堆雪人",
    "If You're Sleepy And You Know It": "如果你困了就打哈欠",
    "Making A Card For My Valentine": "制作情人节卡片",
    "Keep On Swimming": "继续向前游",
    "Super Simple Disco": "欢乐迪斯科",
    "Down On The Plain": "在大平原上",
    "We're Going To The Pumpkin Patch": "我们要去南瓜田",
    "Super Spooky Halloween Storm": "超级吓人的万圣节风暴",
    "Super Simple Thank You": "真诚说谢谢",
    "Christmas Is Almost Here": "圣诞节快到了",
    "We're Going To The Reindeer Games": "我们去参加驯鹿运动会",
    "C-H-R-I-S-T-M-A-S": "圣诞节拼字歌",
    "I Love Shoveling The Snow": "我爱铲雪",
    "We're Walking Down The Street": "我们走在大街上",
    "Hey-O We Want To Play-O": "嘿哟，我们想一起玩",
}


TAG_KEYWORDS = {
    "animals": ("animal", "monkey", "dog", "bingo", "kangaroo", "pet", "spider", "reindeer", "rooster", "alligator", "crocodile", "duck", "frog", "shark", "robin", "camel", "ant", "fish", "bear", "jellyfish", "bee", "butterfly", "ladybug", "kitten", "mouse", "chick", "ghost", "caterpillar", "dinosaur", "crab", "bird", "snail"),
    "numbers": ("count", "one", "two", "three", "four", "five", "six", "seven", "eight", "ten", "twelve", "100", "500", "adding"),
    "colors": ("color", "blue", "pink", "yellow", "green", "red", "purple", "orange", "brown", "rainbow"),
    "holidays": ("christmas", "santa", "reindeer", "jingle", "snowman", "elf", "halloween", "ghost", "monster", "pumpkin", "trick or treat", "valentine", "birthday", "new year", "thankful"),
    "vehicles": ("bus", "car", "truck", "tractor", "airplane", "sailboat", "bicycle", "subway", "rocket", "boat"),
    "bedtime": ("bed", "sleep", "dream", "night", "nap", "hush", "beddy-bye", "tucked"),
    "alphabet": ("alphabet", "abc", "a is for"),
    "movement": ("head shoulders", "finger", "move", "walking", "walk", "dance", "follow me", "stand up", "sit down", "stomp", "left and right", "rock scissors paper", "make a circle", "crawl", "swim"),
    "routines": ("clean up", "brush your teeth", "bath", "put on your", "get dressed", "potty", "wash our hands", "hand washing", "line up", "take turns", "loose tooth"),
    "food": ("broccoli", "ice cream", "banana", "potato", "spaghetti", "yogurt", "cookie", "candy", "pickle", "pudding", "lasagna", "milkshake", "muffin", "apple", "peanut butter", "jelly", "pizza", "strawberry", "milk and cookies", "hungry"),
    "nature": ("weather", "rain", "sun", "snow", "ocean", "sea", "mountain", "forest", "jungle", "grass", "season", "beach", "star", "moon", "hot today", "rainbow"),
    "friends": ("hello", "goodbye", "see you", "good morning", "good night", "friend", "i like you", "i love you", "thank you", "happy", "together", "your name", "birthday"),
}

CONTENT_TAGS = {"animals", "colors", "holidays", "vehicles", "bedtime", "alphabet", "routines", "food", "nature"}


WORD_ZH = {
    "little": "小", "song": "歌", "happy": "快乐", "super": "超级", "simple": "简单",
    "christmas": "圣诞节", "halloween": "万圣节", "house": "房子", "tree": "树", "snow": "雪",
    "baby": "宝宝", "my": "我的", "your": "你的", "you": "你", "me": "我", "we": "我们",
    "love": "爱", "like": "喜欢", "go": "去", "come": "来", "play": "玩", "party": "派对",
    "day": "一天", "morning": "早上", "night": "夜晚", "time": "时间", "dance": "跳舞",
    "walk": "走路", "walking": "走路", "swimming": "游泳", "car": "汽车", "truck": "卡车",
    "fish": "鱼", "ducks": "鸭子", "bird": "小鸟", "dog": "小狗", "animals": "动物",
    "red": "红色", "yellow": "黄色", "green": "绿色", "blue": "蓝色", "pink": "粉色",
    "one": "一", "two": "二", "three": "三", "four": "四", "five": "五", "six": "六",
    "seven": "七", "eight": "八", "ten": "十", "counting": "数数", "count": "数数",
}


def clean_title(value: str) -> str:
    value = re.sub(r"^\s*\d+\s*[. _-]*", "", value)
    value = value.replace("？", "?")
    value = re.sub(r",(s|re|m|t|ve|ll|d)\b", r"'\1", value, flags=re.I)
    return value.strip()


def base_and_suffix(title: str) -> tuple[str, str]:
    suffixes: list[str] = []
    lowered = title.lower()
    if "finny the shark" in lowered:
        suffixes.append("芬尼鲨鱼版")
    elif "noodle" in lowered:
        suffixes.append("面条伙伴版")
    elif "caitie" in lowered:
        suffixes.append("凯蒂版")
    elif "super simple puppets" in lowered:
        suffixes.append("布偶版")
    elif "carl's car wash" in lowered or "carl,s car wash" in lowered:
        suffixes.append("卡尔洗车店版")
    base = re.sub(r"\s+-\s+featuring.*$", "", title, flags=re.I)
    base = re.sub(r"\s+featuring.*$", "", base, flags=re.I)
    base = re.sub(r"\s*\((?:Finny the Shark|Noodle\s*[-&]\s*Pals|Carl['’s, ]+Car Wash|Bumble Nums|Super Simple Puppets|Mr\. Monkey)[^)]*\)\s*$", "", base, flags=re.I)
    version = re.search(r"\s*#(\d+)\s*$", base)
    if version:
        suffixes.append(f"第{version.group(1)}版")
        base = base[: version.start()].strip()
    if "(Part 2)" in base:
        suffixes.append("第二部")
        base = base.replace("(Part 2)", "").strip()
    return base, "" if not suffixes else "（" + "·".join(suffixes) + "）"


def translate_title(title: str) -> str:
    base, suffix = base_and_suffix(title)
    if base in BASE_ZH:
        return BASE_ZH[base] + suffix
    normalized = base.replace("’", "'")
    if normalized in BASE_ZH:
        return BASE_ZH[normalized] + suffix
    words = re.findall(r"[A-Za-z]+|\d+", normalized.lower())
    translated = "".join(WORD_ZH.get(word, word.title()) for word in words)
    return (translated if re.search(r"[\u4e00-\u9fff]", translated) else "趣味儿歌·" + base) + suffix


def contains_phrase(text: str, phrase: str) -> bool:
    return re.search(r"(?<![a-z0-9])" + re.escape(phrase) + r"(?![a-z0-9])", text) is not None


def sql_quote(value: str) -> str:
    return "'" + value.replace("\\", "\\\\").replace("'", "''") + "'"


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("song_dir", type=Path)
    parser.add_argument("--lyrics-dir", type=Path)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()

    rows: list[tuple[int, str, set[str]]] = []
    for path in sorted(args.song_dir.glob("*.mp3")):
        match = re.match(r"\s*(\d+)", path.name)
        if not match:
            continue
        source_no = int(match.group(1))
        title = clean_title(path.stem)
        title_zh = translate_title(title)
        searchable = title.lower().replace("’", "'")
        lyric_text = ""
        if args.lyrics_dir:
            candidates = sorted(args.lyrics_dir.glob(f"{source_no:03d}*.lrc"))
            if candidates:
                lyric_text = candidates[0].read_text(encoding="utf-8", errors="ignore").lower().replace("’", "'")
        tags = {
            slug for slug, phrases in TAG_KEYWORDS.items()
            if any(contains_phrase(searchable + (" " + lyric_text if slug in CONTENT_TAGS else ""), phrase) for phrase in phrases)
        }
        rows.append((source_no, title_zh, tags))

    if len(rows) != 426:
        raise SystemExit(f"expected 426 songs, found {len(rows)}")

    lines = [
        "-- Generated by scripts/generate_catalog_seed.py; contains metadata only, no media resources.",
        "SET NAMES utf8mb4;",
        "INSERT INTO media_catalog (source_no, title_zh) VALUES",
    ]
    values = [f"  ({source_no}, {sql_quote(title_zh)})" for source_no, title_zh, _ in rows]
    lines.append(",\n".join(values))
    lines.extend(["ON DUPLICATE KEY UPDATE title_zh = VALUES(title_zh);", "", "DELETE FROM media_catalog_tag;", ""])
    for slug in TAG_KEYWORDS:
        numbers = [str(source_no) for source_no, _, tags in rows if slug in tags]
        if not numbers:
            continue
        lines.extend([
            "INSERT INTO media_catalog_tag (source_no, tag_id)",
            f"SELECT source_no, (SELECT id FROM media_tag WHERE slug = {sql_quote(slug)})",
            f"FROM media_catalog WHERE source_no IN ({', '.join(numbers)});",
            "",
        ])
    args.output.write_text("\n".join(lines), encoding="utf-8")


if __name__ == "__main__":
    main()
