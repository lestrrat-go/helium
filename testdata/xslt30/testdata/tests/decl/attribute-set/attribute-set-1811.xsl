<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"  
version="2.0">

 <?spec xslt#creating-attributes?>
 
<!-- Test separator on xsl:attribute, with and without select attribute, including default -->
<xsl:template match="/">
     
        <x>
          <xsl:attribute name="a" separator=" ">
            <xsl:sequence select="1, 2, 3"/>
          </xsl:attribute>
          <xsl:attribute name="b" separator="-">
            <xsl:sequence select="1, 2, 3"/>
          </xsl:attribute>
          <xsl:attribute name="c" separator="">
            <xsl:sequence select="1, 2, 3"/>
          </xsl:attribute>
          <xsl:attribute name="d" separator=" " select="1, 2, 3"/>
          <xsl:attribute name="e" separator="-" select="1, 2, 3"/>
          <xsl:attribute name="f" separator="" select="1, 2, 3"/>
          <xsl:attribute name="g">
            <xsl:sequence select="1, 2, 3"/>
          </xsl:attribute>   
          <xsl:attribute name="h" select="1, 2, 3"/>       
        </x>
     
    </xsl:template>

</xsl:stylesheet>
