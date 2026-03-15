<?xml version="1.0" encoding="iso-8859-1"?> 
<xsl:transform version="2.0"
    xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

    <!-- test unparsed-text() with one argument -->

<xsl:output name="test" encoding="iso-8859-1" indent="no"/>
<xsl:template match="/">
  <xsl:result-document format="test">
  <out>
  <xsl:for-each select="'utf-16', 'utf-8'">
    <xsl:variable name="f" select="concat('x', ., '.txt')"/>
    <file name="{$f}"><xsl:value-of select="unparsed-text($f)"/></file>
  </xsl:for-each>
  <xsl:for-each select="'utf-16', 'utf-8', 'iso-8859-1'">  
    <xsl:variable name="g" select="concat('x', ., '.xml')"/>
    <file name="{$g}"><xsl:value-of select="unparsed-text($g)"/></file>    
  </xsl:for-each>
  </out> 
  </xsl:result-document> 
</xsl:template>

<xsl:template name="make">
  <!-- make some sample files in different encodings -->
  <xsl:for-each select="'utf-16', 'utf-8', 'iso-8859-1'">
    <xsl:result-document href="x{.}.txt" method="text" encoding="{.}" byte-order-mark="yes">Kirche, Kinder, Küche</xsl:result-document>
    <xsl:result-document href="x{.}.xml" method="xml"  encoding="{.}" byte-order-mark="yes">Kirche, Kinder, Küche</xsl:result-document>
  </xsl:for-each>
</xsl:template>  

</xsl:transform>