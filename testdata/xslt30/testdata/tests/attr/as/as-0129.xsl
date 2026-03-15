<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test with global xsl:variable that has @as=""document-node()?", 
  				no @select and sequence constructor is empty or has one xsl:document.-->

   <t:variable name="var1" as="document-node()?"> </t:variable>

   <t:variable name="var2" as="document-node()">
      <t:document>
         <item>hello</item>
      </t:document>
   </t:variable>

   <t:template match="/doc">
      <out>
         <e1>
            <t:value-of select="$var1 instance of document-node()?"/>
            <t:value-of select="$var1 instance of document-node()"/>
         </e1>
         <e2>
            <t:value-of select="$var2 instance of document-node()?"/>
            <t:value-of select="$var2 instance of document-node()"/>
         </e2>
      </out>
   </t:template>
</t:transform>
