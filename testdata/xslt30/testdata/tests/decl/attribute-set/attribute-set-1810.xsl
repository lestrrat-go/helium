<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

 <?spec xslt#attribute-sets?>
<!-- test an empty attribute set -->

   <xsl:attribute-set name="a" >
      <!-- <xsl:attribute name="n1" select="s1"/> -->
   </xsl:attribute-set> 

   <xsl:attribute-set name="b" use-attribute-sets="a"/>
   
   <xsl:template match="/">
     <a xsl:use-attribute-sets="b"/>
   </xsl:template>

</xsl:transform>

