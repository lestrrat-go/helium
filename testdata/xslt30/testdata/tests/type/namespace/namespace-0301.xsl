<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0"
                xmlns:a="http://z.com" 
                xmlns:b="http://x.com" 
                xmlns:c="http://y.com">
  <?spec xpath#node-tests?>
  
  <xsl:template match="a">
    <out>
      <xsl:apply-templates select="@a:*">
        <xsl:sort select="namespace-uri(.)"/>
      </xsl:apply-templates>
      <foo><xsl:apply-templates select="@*"/></foo>
    </out>
  </xsl:template>
  
  <xsl:template match="@*">
    <xsl:attribute name="{local-name(.)}">
      <xsl:value-of select="."/>
    </xsl:attribute>
  </xsl:template>
 
</xsl:stylesheet>
