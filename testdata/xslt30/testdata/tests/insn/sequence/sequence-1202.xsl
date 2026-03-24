<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test using xsl:sequence to add a document node to a result tree. -->

   <t:template match="doc">
      <out>
         <t:text>(((</t:text>
         <t:sequence select="document('sequence-1202a.xml')"/>
         <t:text>)))</t:text>
      </out>
   </t:template>
</t:transform>
