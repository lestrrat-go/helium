<?xml version="1.0"?> 

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">

<!-- basic test of idref() function, the argument isn't tokenized like it is for fn:id(), leading in this example to non-matches as the strings are not valid xs:IDREF -->


  <xsl:template match="/">
    <out>
      <xsl:for-each select="//a">
        <xsl:copy>
          <xsl:copy-of select="@*"/>
          <xsl:attribute name="from">
            <xsl:value-of select="for $n in idref('a ' || @id) return $n/../@id" separator=","/>
          </xsl:attribute>
        </xsl:copy>
      </xsl:for-each>
    </out>
  </xsl:template>
 
</xsl:stylesheet>
