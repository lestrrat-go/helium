<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests to verify that the effect of default is only local; the parameter continues 
                in the tunnel in an unset status.-->

   <t:template match="doc">
		    <out>
         <t:next-match>
					
			         <t:with-param name="par1" tunnel="yes"/>
		       </t:next-match>
      </out>
	  </t:template>

   <t:template match="data">
		    <t:param name="par1" select="'123'" tunnel="no"/>		
		    <data xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:text>value of par1 inside data element:</t:text>
         <t:value-of select="$par1"/>
      </data>
		    <t:apply-templates select="*"/>
	  </t:template>

   <t:template match="e2">
		    <t:param name="par1" tunnel="yes"/>
		    <e2 xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <t:text>value of par1 inside e2 element:</t:text>
         <t:value-of select="$par1"/>
      </e2>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
