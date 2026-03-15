<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
    <!-- Purpose: Test for XHTML 5 DOCTYPE declaration.-->
    
    <t:output method="xhtml" html-version="5"/>
    
    <t:template match="/">
        <HtMl xmlns="http://www.w3.org/1999/xhtml">
            <HeAd>
                <Title>test</Title>
            </HeAd>
            <BoDy>
                <P>content</P>
            </BoDy>
        </HtMl>
    </t:template>
</t:transform>
