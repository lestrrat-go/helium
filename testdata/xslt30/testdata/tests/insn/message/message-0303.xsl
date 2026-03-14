<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses a tunnel parameter to set the value of a select attribute
  				in an xsl:message instruction. -->

   <t:template match="/">
		    <out>
         <t:apply-templates select="docs">
				        <t:with-param name="terminate" select="'no'" tunnel="yes"/>
			      </t:apply-templates>
      </out>
	  </t:template>

   <t:template match="docs">
		    <t:param name="terminate" select="'xyz'" tunnel="yes"/>
		    <t:message terminate="{$terminate}">Terminate attribute has a value of: <t:value-of select="$terminate"/>
      </t:message>
	  </t:template>
</t:transform>
