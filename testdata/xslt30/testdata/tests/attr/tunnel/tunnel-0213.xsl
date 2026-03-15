<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test tunnel parameters with attributes tunnel="yes" and tunnel="no"
  				specified within xsl:apply-templates where the first matched element 
  				is the document element. -->

   <t:template match="/">
		    <out>
         <t:apply-templates select="doc/data">
			         <t:with-param name="par1" select="'foo1'" tunnel="yes"/>
			         <t:with-param name="par2" select="'foo2'" tunnel="no"/>
		       </t:apply-templates>
      </out>
	  </t:template>

   <t:template match="data">
		    <t:param name="par1" select="'xyz'" tunnel="yes"/>
		    <t:param name="par2" select="'sos'" tunnel="yes"/>
		    <t:value-of select="$par1"/> and <t:value-of select="$par2"/>
	  </t:template>
</t:transform>
