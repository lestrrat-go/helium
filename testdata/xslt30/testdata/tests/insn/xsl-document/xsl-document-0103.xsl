<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test case that creates a new document using xsl:document in a variable 
  				and uses xsl:copy-of to create a new document from the variable. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:variable name="v">
      <t:document>
         <child/>
      </t:document>
   </t:variable>

   <t:template match="/">
      <t:copy-of select="$v"/>
   </t:template>

   <t:template match="child"/>
</t:transform>
