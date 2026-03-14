<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses xsl:call-template inside an xsl:value-of,
  				the called template itself uses xsl:value of. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:variable name="v" select="'myVar'"/>

   <t:template match="/">
		    <out>
         <t:value-of>
  			       <t:call-template name="temp"/>
  		     </t:value-of>
      </out>
	  </t:template>

   <t:template name="temp">
		    <t:value-of select="$v"/>
	  </t:template>
</t:transform>
