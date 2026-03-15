<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case where launch specifies a named template as initial template 
                and that template gets no tunnel params.-->

   <t:template match="/">
		    <t:call-template name="temp"/>
	  </t:template>

   <t:template name="temp">
		    <t:param name="x" tunnel="yes"/>
		    <out>
         <t:value-of select="$x"/>
      </out>
	  </t:template>
</t:transform>
