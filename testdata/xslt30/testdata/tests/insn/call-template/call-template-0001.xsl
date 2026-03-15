<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Attempt to reference the focus when launched with initial template and 
  				no initial context node specified. -->

   <t:template match="/">
	 <t:comment>ERROR: this value should NOT appear</t:comment>
   </t:template>

   <t:template name="temp">
	  <out>
         <t:copy-of select="ancestor-or-self::*"/>
      </out>
   </t:template>
   
</t:transform>
