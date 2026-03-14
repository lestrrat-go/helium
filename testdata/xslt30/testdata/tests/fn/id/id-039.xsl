<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- basic test of idref() function -->


  <xsl:template match="/">
    <out>
      <xsl:for-each select="//a">
        <xsl:copy>
          <xsl:copy-of select="@*"/>
          <xsl:attribute name="from">
            <xsl:value-of select="for $n in idref(@id) return $n/../@id" separator=","/>
          </xsl:attribute>
        </xsl:copy>
      </xsl:for-each>
    </out>
  </xsl:template>
 
</xsl:stylesheet>
