<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test type of global xsl:variable without @select and @as="document-node()". 
  				Sequence constructor contains an xsl:document instruction. -->

   <t:variable name="var1" as="document-node()">
	     <t:document>
		       <item>hello</item>
	     </t:document>
   </t:variable>

   <t:template match="/doc">
      <out>
         <t:value-of select="$var1 instance of document-node()"/>
      </out>
   </t:template>
</t:transform>
