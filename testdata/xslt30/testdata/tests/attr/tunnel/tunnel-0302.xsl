<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to show tunnel parameters passing through built-in templates. -->

   <t:template match="/">
		    <out>
         <t:apply-templates select="*">
			         <t:with-param name="par1" select="'foo'" tunnel="yes"/>
		       </t:apply-templates>
      </out>
	  </t:template>

   <t:template match="e2">
		    <t:param name="par1" select="'xyz'" tunnel="yes"/>
		Parameter par1 inside e2 element: <t:value-of select="$par1"/>
		    <t:apply-templates/>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
