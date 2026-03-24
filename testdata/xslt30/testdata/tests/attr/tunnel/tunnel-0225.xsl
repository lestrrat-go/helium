<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests the tunnel parameters specified within xsl:next-match with 
       the attribute tunnel="no", tunnel="yes" and also specifying no tunnel attribute.  -->

   <t:template match="/">
		    <out>
         <t:next-match>
			         <t:with-param name="par" select="'abc'" tunnel="yes"/>		
			         <t:with-param name="par2" select="'def'" tunnel="no"/>
			         <t:with-param name="par3" select="'nan'"/>
		       </t:next-match>
      </out>
	  </t:template>

   <t:template match="doc">
		    <out>
         <t:next-match>
			         <t:with-param name="par" select="'max'" tunnel="no"/>
			         <t:with-param name="par2" select="'min'" tunnel="yes"/>
			         <t:with-param name="par3" select="'mod'"/>		
		       </t:next-match>
      </out>
	  </t:template>

   <t:template match="data">
		    <t:param name="par" select="'123'" tunnel="yes"/>
		    <t:param name="par2" select="'999'" tunnel="no"/>
		    <t:param name="par3" select="'765'"/>		
		    <t:value-of select="$par"/> and <t:value-of select="$par2"/> and <t:value-of select="$par3"/>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
