<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test to verify that a dynamic-error is raised if a stylesheet 
  				parameter has a required attribute with no supplied value. -->

   <t:param name="par1" required="yes"/>

   <t:template match="doc">
      <out>
         <t:call-template name="boo"/>
      </out>
   </t:template>

   <t:template name="boo">
	par1= <t:value-of select="$par1"/> 
   </t:template>
</t:transform>
