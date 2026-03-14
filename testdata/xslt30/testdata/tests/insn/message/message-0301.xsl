<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses attribute value template for 'terminate' attribute 
  				of an xsl:message instruction, the value of the AVT is evaluated from 
  				the source document. -->

   <t:template match="/">
      <out>
         <t:message select="$v" terminate="{/docs/terminate[1]}">
</t:message>
      </out>
  </t:template>

   <t:variable name="v">Terminate attribute has a value of: <t:value-of select="/docs/terminate[1]"/>
   </t:variable>
</t:transform>
