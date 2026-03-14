<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case for testing an empty base set using a top-level variable and a top-level parameter. 
 				Both the xsl:variable and xsl:param call a template that is also called from within 
 				xsl:template which has established a base set.-->

   <t:variable name="var"> 
		    <t:call-template name="temp">
			      <t:with-param name="p1" select="insideVar" tunnel="yes"/>
		    </t:call-template>
	  </t:variable>

   <t:param name="param"> 
		    <t:call-template name="temp">
			      <t:with-param name="p1" select="insideparam" tunnel="yes"/>
		    </t:call-template>
	  </t:param>

   <t:template match="doc">
		    <out>
         <t:call-template name="temp">
				        <t:with-param name="p1" select="'ParValue'" tunnel="yes"/>
			      </t:call-template>
      </out>
	  </t:template>

   <t:template name="temp">
		    <t:param name="p1" select="'defValue'" tunnel="yes"/>
		    <t:value-of select="$p1"/>
	  </t:template>
</t:transform>
