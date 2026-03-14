<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: To verify that the Unicode code point collation can be passed as an argument and is used in XPath functions that require collation.-->

   <t:output method="xml" indent="no" encoding="UTF-8"/>

   <t:template match="/">
      <out>
         <Compare>
            <t:value-of select="compare('b', 'ä', 'http://www.w3.org/2005/xpath-functions/collation/codepoint')"/>
         </Compare>
         <Contains>
            <t:value-of select="contains('Strasse', 'ß', 'http://www.w3.org/2005/xpath-functions/collation/codepoint')"/>
         </Contains>
         <Starts-with>
            <t:value-of select="starts-with('Sstrasse', 'ß', 'http://www.w3.org/2005/xpath-functions/collation/codepoint')"/>
         </Starts-with>
         <Ends-with>
            <t:value-of select="ends-with('Strass', 'ß', 'http://www.w3.org/2005/xpath-functions/collation/codepoint')"/>
         </Ends-with>
         <Subtring-before>
            <t:value-of select="substring-before('Strasse', 'raß', 'http://www.w3.org/2005/xpath-functions/collation/codepoint')"/>
         </Subtring-before>
         <Substring-after>
            <t:value-of select="substring-after('Strasse', 'raß', 'http://www.w3.org/2005/xpath-functions/collation/codepoint')"/>
         </Substring-after>
         <Index-of>
            <t:value-of select="index-of(('SS', 's', 'sse', 'ss'), 'ß', 'http://www.w3.org/2005/xpath-functions/collation/codepoint')"/>
         </Index-of>
         <Distinct-values>
            <t:value-of select="distinct-values(('Strasse', 'Straß'), 'http://www.w3.org/2005/xpath-functions/collation/codepoint')"/>
         </Distinct-values>
         <Deep-equal>
            <t:value-of select="deep-equal('Strasse', 'Straße', 'http://www.w3.org/2005/xpath-functions/collation/codepoint')"/>
         </Deep-equal>
         <Max>
            <t:value-of select="max(('z', 'ä'), 'http://www.w3.org/2005/xpath-functions/collation/codepoint')"/>
         </Max>
         <Min>
            <t:value-of select="max(('z', 'ä'), 'http://www.w3.org/2005/xpath-functions/collation/codepoint')"/>
         </Min>
      </out>
   </t:template>
</t:transform>
