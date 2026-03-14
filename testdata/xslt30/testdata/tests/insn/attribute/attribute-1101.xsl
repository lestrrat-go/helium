<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <?spec xslt#creating-attributes?>
    <!-- Purpose: Verify that attributes that contain text nodes with 
       a newline, the output must contain a character reference. -->

<xsl:template match="/">
   <Out>
      <xsl:attribute name="attr1">x
	  y</xsl:attribute>
   </Out>
</xsl:template>

</xsl:stylesheet>